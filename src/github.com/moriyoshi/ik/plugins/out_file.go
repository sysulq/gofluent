package plugins

import (
	"github.com/moriyoshi/ik"
	jnl "github.com/moriyoshi/ik/journal"
	"io"
	"strconv"
	"math/rand"
	"compress/gzip"
	"os"
	"fmt"
	"log"
	"time"
	"path"
	"errors"
	"strings"
	"encoding/json"
	strftime "github.com/jehiah/go-strftime"
)

type FileOutput struct {
	factory *FileOutputFactory
	logger  *log.Logger
	pathPrefix string
	pathSuffix string
	symlinkPath string
	permission os.FileMode
	compressionFormat int
	journalGroup ik.JournalGroup
	slicer *ik.Slicer
	timeFormat string
	timeSliceFormat string
	location *time.Location
	c chan []ik.FluentRecordSet
	cancel chan bool
	disableDraining bool
}

type FileOutputPacker struct {
	output *FileOutput
}

type FileOutputFactory struct {
}

const (
	compressionNone = 0
	compressionGzip = 1
)

func (packer *FileOutputPacker) Pack(record ik.FluentRecord) ([]byte, error) {
	formattedData, err := packer.output.formatData(record.Data)
	if err != nil {
		return nil, err
	}
	return ([]byte)(fmt.Sprintf(
		"%s\t%s\t%s\n",
		packer.output.formatTime(record.Timestamp),
		record.Tag,
		formattedData,
	)), nil
}

func (output *FileOutput) formatTime(timestamp uint64) string {
	timestamp_ := time.Unix(int64(timestamp), 0)
	if output.timeFormat == "" {
		return timestamp_.Format(time.RFC3339)
	} else {
		return strftime.Format(output.timeFormat, timestamp_)
	}
}

func (output *FileOutput) formatData(data map[string]interface{}) (string, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(b), nil // XXX: byte => rune
}

func (output *FileOutput) Emit(recordSets []ik.FluentRecordSet) error {
	output.c <- recordSets
	return nil
}

func (output *FileOutput) Factory() ik.Plugin {
	return output.factory
}

func (output *FileOutput) Run() error {
	select {
	case <- output.cancel:
		return nil
	case recordSets := <-output.c:
		err := output.slicer.Emit(recordSets)
		if err != nil {
			return err
		}
	}
	return ik.Continue
}

func (output *FileOutput) Shutdown() error {
	output.cancel <- true
	return output.journalGroup.Dispose()
}

func (output *FileOutput) Dispose() {
	output.Shutdown()
}

func buildNextPathName(key string, pathPrefix string, pathSuffix string, suffix string) (string, error) {
	i := 0
	var path_ string
	for {
		path_ = fmt.Sprintf(
			"%s%s_%d%s%s",
			pathPrefix,
			key,
			i,
			pathSuffix,
			suffix,
		)
		_, err := os.Stat(path_)
		if err != nil {
			if os.IsNotExist(err) {
				err = nil
				break
			} else {
				return "", err
			}
		}
		i += 1
	}
	dir := path.Dir(path_)
	err := os.MkdirAll(dir, os.FileMode(os.ModePerm))
	if err != nil {
		return "", err
	}
	return path_, nil
}

func (output *FileOutput) flush(key string, chunk ik.JournalChunk) error {
	suffix := ""
	if output.compressionFormat == compressionGzip {
		suffix = ".gz"
	}
	outPath, err := buildNextPathName(
		key,
		output.pathPrefix,
		output.pathSuffix,
		suffix,
	)
	if err != nil {
		return err
	}
	var writer io.WriteCloser
	writer, err = os.OpenFile(outPath, os.O_CREATE | os.O_EXCL | os.O_WRONLY, output.permission)
	if err != nil {
		return err
	}
	defer writer.Close()

	if output.compressionFormat == compressionGzip {
		writer = gzip.NewWriter(writer)
		defer writer.Close()
	}

	reader, err := chunk.GetReader()
	closer, _ := reader.(io.Closer)
	if closer != nil {
		defer closer.Close()
	}
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, reader)
	if err != nil {
		return err
	}
	return nil
}

func (output *FileOutput) attachListeners(journal ik.Journal) {
	if output.symlinkPath != "" {
		journal.AddNewChunkListener(func (chunk ik.JournalChunk) error {
			defer chunk.Dispose()
			wrapper, ok := chunk.(*jnl.FileJournalChunkWrapper)
			if !ok {
				return nil
			}
			err := os.Remove(output.symlinkPath)
			if err != nil && !os.IsNotExist(err) {
				return err
			} else {
				err = os.Symlink(output.symlinkPath, wrapper.Path())
			}
			if err != nil {
				output.logger.Print("Failed to create symbolic link " + output.symlinkPath)
			}
			return err
		})
	}
	journal.AddFlushListener(func (chunk ik.JournalChunk) error {
		defer chunk.Dispose()
		chunk.TakeOwnership()
		return output.flush(journal.Key(), chunk)
	})
}

func newFileOutput(factory *FileOutputFactory, logger *log.Logger, randSource rand.Source, pathPrefix string, pathSuffix string, timeFormat string, compressionFormat int, symlinkPath string, permission os.FileMode, bufferChunkLimit int64, timeSliceFormat string, disableDraining bool) (*FileOutput, error) {
	if timeSliceFormat == "" {
		timeSliceFormat = "%Y%m%d"
	}
	journalGroupFactory := jnl.NewFileJournalGroupFactory(
		logger,
		randSource,
		func () time.Time { return time.Now() },
		pathSuffix,
		permission,
		bufferChunkLimit,
	)
	retval := &FileOutput {
		factory: factory,
		logger:  logger,
		pathPrefix: pathPrefix,
		pathSuffix: pathSuffix,
		symlinkPath: symlinkPath,
		permission: permission,
		compressionFormat: compressionFormat,
		timeFormat: timeFormat,
		timeSliceFormat: timeSliceFormat,
		location: time.UTC,
		c: make(chan []ik.FluentRecordSet, 100 /* FIXME */),
		cancel: make(chan bool),
		disableDraining: disableDraining,
	}
	journalGroup, err := journalGroupFactory.GetJournalGroup(pathPrefix, retval)
	if err != nil {
		return nil, err
	}

	slicer := ik.NewSlicer(
		journalGroup,
		func (record ik.FluentRecord) string {
			timestamp_ := time.Unix(int64(record.Timestamp), 0)
			return strftime.Format(retval.timeSliceFormat, timestamp_)
		},
		&FileOutputPacker { retval },
		logger,
	)
	slicer.AddNewKeyEventListener(func (last ik.Journal, next ik.Journal) error {
		err := (error)(nil)
		if last != nil {
			err = last.Flush(func (chunk ik.JournalChunk) error {
				defer chunk.Dispose()
				chunk.TakeOwnership()
				return retval.flush(last.Key(), chunk)
			})
		}
		if next != nil {
			if !retval.disableDraining {
				retval.attachListeners(next)
			}
		}
		return err
	})
	retval.journalGroup = journalGroup
	retval.slicer = slicer
	if !disableDraining {
		currentKey := strftime.Format(timeSliceFormat, time.Now())
		for _, key := range journalGroup.GetJournalKeys() {
			if key == currentKey {
				journal := journalGroup.GetJournal(key)
				retval.attachListeners(journal)
				journal.Flush(nil)
			}
		}
	}
	return retval, nil
}

func (factory *FileOutputFactory) Name() string {
	return "file"
}

func (factory *FileOutputFactory) New(engine ik.Engine, config *ik.ConfigElement) (ik.Output, error) {
	pathPrefix := ""
	pathSuffix := ""
	timeFormat := ""
	compressionFormat := compressionNone
	symlinkPath := ""
	permission := 0666
	bufferChunkLimit := int64(8 * 1024 * 1024) // 8MB
	timeSliceFormat := ""
	disableDraining := false

	path, ok := config.Attrs["path"]
	if !ok {
		return nil, errors.New("'path' parameter is required on file output")
	}
	timeFormat, _ = config.Attrs["time_format"]
	compressionFormatStr, ok := config.Attrs["compress"]
	if ok {
		if compressionFormatStr == "gz" || compressionFormatStr == "gzip" {
			compressionFormat = compressionGzip
		} else {
			return nil, errors.New("unknown compression format: " + compressionFormatStr)
		}
	}
	symlinkPath, _ = config.Attrs["symlink_path"]
	permissionStr, ok := config.Attrs["permission"]
	if ok {
		var err error
		permission, err = strconv.Atoi(permissionStr)
		if err != nil {
			return nil, err
		}
	}
	pos := strings.Index(path, "*")
	if pos >= 0 {
		pathPrefix = path[0:pos]
		pathSuffix = path[pos + 1:]
	} else {
		pathPrefix = path + "."
		pathSuffix = ".log"
	}
	timeSliceFormat, _ = config.Attrs["time_slice_format"]

	bufferChunkLimitStr, ok := config.Attrs["buffer_chunk_limit"]
	if ok {
		var err error
		bufferChunkLimit, err = ik.ParseCapacityString(bufferChunkLimitStr)
		if err != nil {
			return nil, err
		}
	}

	disableDrainingStr, ok := config.Attrs["disable_draining"]
	if ok {
		var err error
		disableDraining, err = strconv.ParseBool(disableDrainingStr)
		if err != nil {
			return nil, err
		}
	}

	return newFileOutput(
		factory,
		engine.Logger(),
		engine.RandSource(),
		pathPrefix,
		pathSuffix,
		timeFormat,
		compressionFormat,
		symlinkPath,
		os.FileMode(permission),
		bufferChunkLimit,
		timeSliceFormat,
		disableDraining,
	)
}

func (factory *FileOutputFactory) BindScorekeeper(scorekeeper *ik.Scorekeeper) {
}

var _ = AddPlugin(&FileOutputFactory{})
