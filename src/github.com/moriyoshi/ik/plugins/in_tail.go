package plugins

import (
	"os"
	"io"
	"fmt"
	"log"
	"time"
	"bytes"
	"errors"
	"strconv"
	"strings"
	"reflect"
	"net/http"
	"encoding/hex"
	"sync"
	"github.com/moriyoshi/ik"
	fileid "github.com/moriyoshi/go-fileid"
	"github.com/howeyc/fsnotify"
)

const DefaultBacklogSize = 2048

type MyBufferedReader struct {
	inner io.Reader
	b []byte
	r, w int
	position int64
	eofReached bool
}

func (b *MyBufferedReader) shift() {
	copy(b.b, b.b[b.r:b.w])
	b.w -= b.r
	b.r = 0
}

func (b *MyBufferedReader) fillUp() (bool, error) {
	if b.eofReached {
		return false, nil
	}
	if b.w == len(b.b) {
		return false, nil
	}
	n, err := b.inner.Read(b.b[b.w:])
	if err != nil {
		if err == io.EOF {
			b.eofReached = true
			return true, nil
		} else {
			return false, err
		}
	}
	b.w += n
	return true, nil
}

func (b *MyBufferedReader) Reset(position int64) error {
	b.position = position
	b.r = 0
	b.w = 0
	return nil
}

func (b *MyBufferedReader) Continue() {
	b.eofReached = false
}

func (b *MyBufferedReader) ReadRest() []byte {
	if !b.eofReached {
		return nil
	}
	line := b.b[b.r:b.w]
	b.r = b.w
	return line
}

func (b *MyBufferedReader) ReadLine() ([]byte, bool, bool, error) {
	for {
		i := bytes.IndexByte(b.b[b.r:b.w], '\n')
		if i >= 0 {
			e := b.r + i
			if e > 0 && b.b[e - 1] == '\r' {
				e -= 1
			}
			line := b.b[b.r:e]
			b.position += int64(i + 1)
			b.r += i + 1
			return line, false, false, nil
		}

		if b.eofReached {
			if b.r == b.w {
				return b.b[b.r:b.r], false, false, io.EOF
			} else {
				return b.b[b.r:b.r], false, true, nil
			}
		}

		b.shift()
		more, err := b.fillUp()
		if err != nil {
			return nil, false, false, err
		}

		if !more {
			break
		}
	}
	{
		line := b.b[b.r:]
		b.r = 0
		b.w = 0
		b.position += int64(len(line))
		return line, true, false, nil
	}
}

func NewMyBufferedReader(reader io.Reader, bufSize int, position int64) *MyBufferedReader {
	return &MyBufferedReader {
		inner: reader,
		b: make([]byte, bufSize),
		r: 0,
		w: 0,
		position: position,
		eofReached: false,
	}
}

type TailTarget struct {
	path string
	f *os.File
	size int64
	id fileid.FileId
}

type TailEventHandler struct {
	logger *log.Logger
	path string
	rotateWait time.Duration
	target TailTarget
	pending bool
	expiry time.Time
	bf *MyBufferedReader
	readBufferSize int
	decoder func ([]byte) (string, error)
	stateSaver func (target TailTarget, position int64) error
	lineReceiver func (line string) error
	closer func () error
}

func openTarget(path string) (TailTarget, error) {
	f, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		if os.IsNotExist(err) {
			return TailTarget {
				path: path,
				f: nil,
			}, nil
		} else {
			return TailTarget {}, err
		}
	}

	info, err := f.Stat()
	if err != nil {
		return TailTarget {}, err
	}

	id, err := fileid.GetFileId(path, true)
	if err != nil {
		return TailTarget {}, err
	}

	return TailTarget {
		path: path,
		f: f,
		size: info.Size(),
		id: id,
	}, nil
}

func (target *TailTarget) UpdatedOne() (TailTarget, error) {
	info, err := target.f.Stat()
	if err != nil {
		return TailTarget {}, err
	}
	return TailTarget {
		path: target.path,
		f: target.f,
		size: info.Size(),
		id: target.id,
	}, nil
}

func (handler *TailEventHandler) decode(in []byte) (string, error) {
	if handler.decoder != nil {
		return handler.decoder(in)
	} else {
		return string(in), nil
	}
}

func (handler *TailEventHandler) fetch() error {
	for {
		line, ispfx, tryAgain, err := handler.bf.ReadLine()
		if err != nil {
			if err == io.EOF {
				handler.bf.Continue()
				break
			} else {
				return err
			}
		}
		if tryAgain {
			if !handler.pending {
				break
			} else {
				line = handler.bf.ReadRest()
				if line == nil {
					panic("WTF?")
				}
				handler.logger.Printf("the file was not terminated by line endings and the file seems to have been rotated: %s, position=%d",  handler.target.path, handler.bf.position)
			}
		}
		if ispfx {
			handler.logger.Printf("line too long: %s, position=%d",  handler.target.path, handler.bf.position)
		}
		stringizedLine, err := handler.decode(line)
		if err != nil {
			handler.logger.Printf("failed to decode line")
			stringizedLine = hex.Dump(line)
		}
		err = handler.lineReceiver(stringizedLine)
		if err != nil {
			return err
		}
	}
	return nil
}

func (handler *TailEventHandler) saveState() error {
	return handler.stateSaver(
		handler.target,
		handler.bf.position,
	)
}

func (handler *TailEventHandler) Dispose() error {
	var err error
	if handler.target.f != nil {
		err = handler.target.f.Close()
		handler.target.f = nil
	}
	if handler.closer != nil {
		err_ := handler.closer()
		if err_ != nil {
			err = err_
		}
	}
	return err
}

func (handler *TailEventHandler) OnChange(now time.Time) error {
	var target TailTarget
	var err error

	if handler.pending {
		// there is pending rotation request
		if now.Sub(handler.expiry) >= 0 {
			// target was expired
			target, err = openTarget(handler.path)
			if err != nil {
				return err
			}
			handler.pending = false
		} else {
			target, err = (&handler.target).UpdatedOne()
			if err != nil {
				return err
			}
		}
	} else {
		target, err = openTarget(handler.path)
		if err != nil {
			return err
		}
		if handler.target.f != nil && target.f != nil {
			sameFile := fileid.IsSame(handler.target.id, target.id)
			err := target.f.Close()
			if err != nil {
				return err
			}
			if !sameFile {
				// file was replaced
				target, err = (&handler.target).UpdatedOne()
				if err != nil {
					return err
				}
				handler.logger.Println("rotation detected: " + target.path)
				handler.pending = true
				handler.expiry = now.Add(handler.rotateWait)
			} else {
				target.f = handler.target.f
			}
		} else if handler.target.f == nil && target.f != nil {
			// file was newly created
			handler.logger.Println("file created: " + target.path)
		} else if handler.target.f != nil && target.f == nil {
			// file was moved?
			handler.logger.Println("file moved: " + target.path)
			target, err = (&handler.target).UpdatedOne()
			if err != nil {
				return err
			}
			handler.pending = true
			handler.expiry = now.Add(handler.rotateWait)
		}
	}

	fetchNeeded := false
	if target.f != handler.target.f || target.size < handler.target.size {
		// file was replaced / moved / created / truncated
		newPosition := target.size
		if target.f != handler.target.f {
			if handler.target.f != nil {
				err = handler.target.f.Close()
				if err != nil {
					return err
				}
			}
			newPosition = 0
		}
		position, err := target.f.Seek(newPosition, os.SEEK_SET)
		if err != nil {
			return err
		}
		handler.bf = NewMyBufferedReader(target.f, handler.readBufferSize, position)
		fetchNeeded = true
	} else {
		fetchNeeded = target.size > handler.target.size
	}
	handler.target = target

	if fetchNeeded {
		err = handler.fetch()
		if err != nil {
			return err
		}
		err = handler.saveState()
		if err != nil {
			return err
		}
	}
	return nil
}

func NewTailEventHandler(
	logger *log.Logger,
	target TailTarget,
	position int64,
	rotateWait time.Duration,
	readBufferSize int,
	decoder func ([]byte) (string, error),
	stateSaver func (target TailTarget, position int64) error,
	lineReceiver func (line string) error,
	closer func () error,
) (*TailEventHandler, error) {
	bf := (*MyBufferedReader)(nil)
	if target.f != nil {
		position, err := target.f.Seek(position, os.SEEK_SET)
		if err != nil {
			target.f.Close()
			return nil, err
		}
		bf = NewMyBufferedReader(target.f, readBufferSize, position)
		target.size = position
	} else {
		logger.Println("file does not exist: " + target.path)
	}
	return &TailEventHandler {
		logger: logger,
		path: target.path,
		rotateWait: rotateWait,
		target: target,
		pending: false,
		expiry: time.Time {},
		bf: bf,
		readBufferSize: readBufferSize,
		decoder: decoder,
		stateSaver: stateSaver,
		lineReceiver: lineReceiver,
		closer: closer,
	}, nil
}

type TailFileInfo interface {
	GetPosition() int64
	SetPosition(int64)
	GetFileId() fileid.FileId
	SetFileId(fileid.FileId)
	Path() string
	IsNew() bool
	Save() error
	Duplicate() TailFileInfo
	Dispose() error
}

type tailPositionFileData struct {
	Path     string
	Position int64
	Id       fileid.FileId
}

type TailPositionFileEntry struct {
	positionFile *TailPositionFile
	offset       int
	isNew        bool
	data         tailPositionFileData
}

type TailPositionFile struct {
	logger    *log.Logger
	refcount  int64
	path      string
	f         *os.File
	view      []byte
	entries   map[string]*TailPositionFileEntry
	controlChan chan bool
	mtx       sync.Mutex
}

type concreateTailFileInfo struct {
	disposed bool
	impl *TailPositionFileEntry
}

func (wrapper *concreateTailFileInfo) Path() string{
	if wrapper.disposed {
		panic("already disposed")
	}
	return wrapper.impl.data.Path
}

func (wrapper *concreateTailFileInfo) IsNew() bool {
	if wrapper.disposed {
		panic("already disposed")
	}
	return wrapper.impl.isNew
}

func (wrapper *concreateTailFileInfo) GetPosition() int64 {
	if wrapper.disposed {
		panic("already disposed")
	}
	return wrapper.impl.data.Position
}

func (wrapper *concreateTailFileInfo) SetPosition(position int64) {
	if wrapper.disposed {
		panic("already disposed")
	}
	wrapper.impl.data.Position = position
}

func (wrapper *concreateTailFileInfo) GetFileId() fileid.FileId {
	if wrapper.disposed {
		panic("already disposed")
	}
	return wrapper.impl.data.Id
}

func (wrapper *concreateTailFileInfo) SetFileId(id fileid.FileId) {
	if wrapper.disposed {
		panic("already disposed")
	}
	wrapper.impl.data.Id = id
}

func (wrapper *concreateTailFileInfo) Duplicate() TailFileInfo {
	if wrapper.disposed {
		panic("already disposed")
	}
	return newConcreteTailFileInfo(wrapper.impl)
}

func (wrapper *concreateTailFileInfo) Save() error {
	if wrapper.disposed {
		panic("already disposed")
	}
	return wrapper.impl.save()
}

func (wrapper *concreateTailFileInfo) Dispose() error {
	if wrapper.disposed {
		panic("already disposed")
	}
	wrapper.disposed = true
	return wrapper.impl.deleteRef()
}

func newConcreteTailFileInfo(impl *TailPositionFileEntry) *concreateTailFileInfo {
	impl.addRef()
	return &concreateTailFileInfo {
		disposed: false,
		impl: impl,
	}
}

func appendZeroPaddedUint(b []byte, value uint64, size uintptr) []byte {
	fieldLen := 0
	switch size {
	case 1:
		fieldLen = 3
	case 2:
		fieldLen = 5
	case 4:
		fieldLen = 10
	case 8:
		fieldLen = 20
	default:
		panic("WTF?")
	}
	o := len(b)
	requiredCap := o + fieldLen
	if requiredCap > cap(b) {
		nb := make([]byte, requiredCap)
		copy(nb, b)
		b = nb
	}
	b = b[0:requiredCap]
	i := o + fieldLen
	for {
		i -= 1
		if i < o {
			break
		}
		b[i] = byte('0') + byte(value % 10)
		value /= 10
		if value == 0 {
			break
		}
	}
	for {
		i -= 1
		if i < o {
			break
		}
		b[i] = '0'
	}
	return b
}

func marshalPositionFileData(data *tailPositionFileData) []byte {
	b := make([]byte, 0, 16)
	b = append(b, data.Path...)
	b = append(b, '\t')
	b = appendZeroPaddedUint(b, uint64(data.Position), 8)
	rv := reflect.ValueOf(data.Id)
	n := rv.NumField()
	for i := 0; i < n; i += 1 {
		v := rv.Field(i)
		b = append(b, '\t')
		t := v.Type()
		switch t.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			b = appendZeroPaddedUint(b, uint64(v.Int()), t.Size())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			b = appendZeroPaddedUint(b, v.Uint(), t.Size())
		}
	}
	b = append(b, '\n')
	return b
}

func parseUint(b []byte) (uint64, int, error) {
	i := 0
	value := (uint64)(0)
	for i < len(b) && b[i] >= '0' && b[i] <= '9' {
		d := b[i] - '0'
		value *= 10
		value += uint64(d)
		i += 1
	}
	if i == 0 {
		return 0, 0, errors.New("no digits")
	}
	return value, i, nil
}

func unmarshalPositionFileData(retval *tailPositionFileData, blob []byte) (int, error) {
	i := bytes.IndexByte(blob, '\t')
	if i < 0 {
		return 0, errors.New("invalid format")
	}
	retval.Path = string(blob[0:i])

	i += 1
	if i >= len(blob) {
		return 0, errors.New("unexpected EOF")
	}
	{
		v, e, err := parseUint(blob[i:])
		if err != nil {
			return 0, err
		}
		i += e
		retval.Position = int64(v)
	}

	rv := reflect.ValueOf(&retval.Id).Elem()
	fi := 0
	n := rv.NumField()
	outer: for {
		switch blob[i] {
		case '\t':
			i += 1
		case '\n':
			i += 1
			break outer
		default:
			return 0, errors.New("unexpected character: " + string(rune(blob[i])))
		}
		if fi >= n {
			return 0, errors.New("too many fields")
		}
		if i >= len(blob) {
			return 0, errors.New("unexpected EOF")
		}
		v, e, err := parseUint(blob[i:])
		if err != nil {
			return 0, err
		}
		f := rv.Field(fi)
		switch f.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			f.SetInt(int64(v))
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			f.SetUint(v)
		}
		i += e
		fi += 1
	}
	return i, nil
}

func (entry *TailPositionFileEntry) save() error {
	entry.isNew = false
	return entry.positionFile.scheduleUpdate(entry)
}

func (entry *TailPositionFileEntry) addRef() {
	entry.positionFile.addRef()
}

func (entry *TailPositionFileEntry) deleteRef() error {
	return entry.positionFile.deleteRef()
}

func (positionFile *TailPositionFile) scheduleUpdate(entry *TailPositionFileEntry) error {
	blob := marshalPositionFileData(&entry.data)
	offset := entry.offset
	copy(positionFile.view[offset:offset + len(blob)], blob)
	positionFile.controlChan <- false
	return nil
}

func (positionFile *TailPositionFile) doUpdate() {
	for needsToBeStopped := range positionFile.controlChan {
		if needsToBeStopped {
			break
		}
		_, err := positionFile.f.Seek(0, os.SEEK_SET)
		if err != nil {
			positionFile.logger.Println(err.Error())
			continue
		}
		n, err := positionFile.f.Write(positionFile.view)
		if err != nil {
			positionFile.logger.Println(err.Error())
			continue
		}
		if n != len(positionFile.view) {
			positionFile.logger.Println("failed to update position file")
			continue
		}
	}
}

func (positionFile *TailPositionFile) Get(path string) TailFileInfo {
	positionFile.mtx.Lock()
	defer positionFile.mtx.Unlock()
	entry, ok := positionFile.entries[path]
	if !ok {
		offset := len(positionFile.view)
		entry = &TailPositionFileEntry {
			positionFile: positionFile,
			offset:       offset,
			isNew:        true,
			data:         tailPositionFileData {
				Path:     path,
				Position: 0,
				Id:       fileid.FileId {},
			},
		}
		blob := marshalPositionFileData(&entry.data)
		newView := make([]byte, offset + len(blob))
		copy(newView, positionFile.view)
		copy(newView[offset:], blob)
		positionFile.view = newView
		positionFile.entries[path] = entry
	}

	return newConcreteTailFileInfo(entry)
}

func (positionFile *TailPositionFile) Dispose() error {
	return positionFile.deleteRef()
}

func readTailPositionFileEntries(positionFile *TailPositionFile, f *os.File) (map[string]*TailPositionFileEntry, error) {
	retval := make(map[string]*TailPositionFileEntry)
	_, err := f.Seek(0, os.SEEK_SET)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	blob := make([]byte, int(info.Size()))
	n, err := f.Read(blob)
	if err != nil {
		return nil, err
	}
	if n != len(blob) {
		return nil, errors.New("unexpected EOF")
	}
	consumed := 0
	for offset := 0; offset < len(blob); offset += consumed {
		entry := &TailPositionFileEntry {
			positionFile: positionFile,
			offset:       offset,
			isNew:        false,
			data:         tailPositionFileData {},
		}
		consumed, err = unmarshalPositionFileData(&entry.data, blob[offset:])
		if err != nil {
			return nil, err
		}
		retval[entry.data.Path] = entry
	}
	positionFile.view = blob
	return retval, nil
}

func openPositionFile(logger *log.Logger, path string) (*TailPositionFile, error) {
	f, err := os.OpenFile(path, os.O_RDWR | os.O_CREATE | os.O_EXCL, 0666) // FIXME
	failed := true
	defer func () {
		if failed && f != nil {
			f.Close()
		}
	}()
	var entries map[string]*TailPositionFileEntry
	retval := &TailPositionFile {
		logger: logger,
		refcount: 1,
		path: path,
		controlChan: make(chan bool),
		mtx: sync.Mutex {},
	}
	if err != nil {
		if !os.IsExist(err) {
			return nil, err
		}
		f, err = os.OpenFile(path, os.O_RDWR, 0666)
		if err != nil {
			return nil, err
		}
		entries, err = readTailPositionFileEntries(retval, f)
		if err != nil {
			return nil, err
		}
	} else {
		entries = make(map[string]*TailPositionFileEntry)
	}
	retval.f = f
	retval.entries = entries
	failed = false
	go retval.doUpdate()
	return retval, nil
}

func (positionFile *TailPositionFile) save() error {
	_, err := positionFile.f.Seek(0, os.SEEK_SET)
	if err != nil {
		return err
	}
	l, err := positionFile.f.Write(positionFile.view)
	if err != nil {
		return err
	}
	if l != len(positionFile.view) {
		return errors.New("marshalled data not fully written")
	}
	return nil
}

func (positionFile *TailPositionFile) addRef() {
	positionFile.refcount += 1
}

func (positionFile *TailPositionFile) deleteRef() error {
	positionFile.refcount -= 1
	if positionFile.refcount == 0 {
		err := positionFile.f.Close()
		if err != nil {
			positionFile.refcount += 1 // undo the change
			return err
		}
		positionFile.f = nil
		positionFile.controlChan <- true
		close(positionFile.controlChan)
	} else if positionFile.refcount < 0 {
		panic("refcount < 0!")
	}
	return nil
}


type TailInputFactory struct {
}

type TailWatcher struct {
	input             *TailInput
	synthesizedTag    string
	positionFile      *TailPositionFile
	tailFileInfo      TailFileInfo
	lineParser        ik.LineParser
	handler           *TailEventHandler
	statWatcher       *fsnotify.Watcher
	timer             *time.Ticker
	controlChan       chan bool
}

func (watcher *TailWatcher) cleanup() {
	if watcher.statWatcher != nil {
		watcher.statWatcher.Close()
		watcher.statWatcher = nil
	}
	if watcher.handler != nil {
		watcher.handler.Dispose()
		watcher.handler = nil
	}
	if watcher.timer != nil {
		watcher.timer.Stop()
		watcher.timer = nil
	}
}

func (watcher *TailWatcher) Run() error {
	for {
		select {
		case err := <-watcher.statWatcher.Error:
			watcher.input.logger.Println(err.Error())
		case <-watcher.statWatcher.Event:
			now := time.Now()
			err := watcher.handler.OnChange(now)
			if err != nil {
				watcher.input.logger.Println(err.Error())
				return err
			}
			return ik.Continue
		case now := <-watcher.timer.C:
			err := watcher.handler.OnChange(now)
			if err != nil {
				watcher.input.logger.Println(err.Error())
				return err
			}
			return ik.Continue
		case needsToBeStopped := <-watcher.controlChan:
			if needsToBeStopped {
				break
			}
			now := time.Now()
			err := watcher.handler.OnChange(now)
			if err != nil {
				watcher.input.logger.Println(err.Error())
				return err
			}
			return ik.Continue
		}
	}
	watcher.cleanup()
	return nil
}

func (watcher *TailWatcher) Shutdown() error {
	watcher.controlChan <- true
	return nil
}

func (watcher *TailWatcher) parseLine(line string) error {
	return watcher.lineParser.Feed(line)
}

func buildTagFromPath(path string) string {
	b := make([]byte, 0, len(path))
	state := 0
	for i := 0; i < len(path); i += 1 {
		c := path[i]
		if c == '/' || c == '/'{
			state = 1
		} else {
			if state == 1 {
				b = append(b, '.')
				state = 0
			}
			b = append(b, c)
		}
	}
	if state == 1 {
		b = append(b, '.')
	}
	if b[0] == '.' {
		b = b[1:]
	}
	return string(b)
}

func (input *TailInput) openTailWatcher(path string) (*TailWatcher, error) {
	_, ok := input.watchers[path]
	if ok {
		return nil, errors.New(fmt.Sprintf("watcher for path %s already exists", path))
	}
	tailFileInfo := input.positionFile.Get(path)
	defer tailFileInfo.Dispose()
	target, err := openTarget(path)
	if err != nil {
		if os.IsNotExist(err) {
			target = TailTarget { path, nil, 0, fileid.FileId {} }
		} else {
			return nil, err
		}
	}
	if tailFileInfo.IsNew() {
		tailFileInfo.SetFileId(target.id)
		tailFileInfo.SetPosition(target.size)
		err = tailFileInfo.Save()
		if err != nil {
			return nil, err
		}
	}
	watcher := &TailWatcher {
		input:          input,
		synthesizedTag: buildTagFromPath(path),
	}
	handler, err := NewTailEventHandler(
		input.logger,
		target,
		tailFileInfo.GetPosition(),
		input.rotateWait,
		input.readBufferSize,
		nil, // decoder
		func (target TailTarget, position int64) error {
			tailFileInfo := watcher.tailFileInfo
			tailFileInfo.SetFileId(target.id)
			tailFileInfo.SetPosition(position)
			return tailFileInfo.Save()
		},
		func (line string) error {
			return watcher.parseLine(line)
		},
		func () error {
			return watcher.tailFileInfo.Dispose()
		},
	)
	if err != nil {
		return nil, err
	}
	lineParser, err := input.lineParserFactory.New(func (record ik.FluentRecord) error {
		record.Tag = input.tagPrefix
		input.pump.EmitOne(record)
		return nil
	})
	if err != nil {
		handler.Dispose()
		return nil, err
	}
	newStatWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		handler.Dispose()
		return nil, err
	}
	err = newStatWatcher.Watch(path)
	if err != nil {
		newStatWatcher.Close()
		handler.Dispose()
		return nil, err
	}

	watcher.input        = input
	watcher.tailFileInfo = tailFileInfo.Duplicate()
	watcher.handler      = handler
	watcher.statWatcher  = newStatWatcher
	watcher.lineParser   = lineParser
	watcher.timer        = time.NewTicker(time.Duration(1000000000)) // XXX
	watcher.controlChan  = make(chan bool, 1)

	err = input.engine.Spawn(watcher)
	if err != nil {
		watcher.cleanup()
	}

	watcher.controlChan <- false
	return watcher, nil
}

type TailInput struct {
	factory           *TailInputFactory
	engine            ik.Engine
	port              ik.Port
	logger            *log.Logger
	pathSet           *PathSet
	tagPrefix         string
	tagSuffix         string
	rotateWait        time.Duration
	readFromHead      bool
	refreshInterval   time.Duration
	readBufferSize    int
	lineParserFactory ik.LineParserFactory
	positionFile      *TailPositionFile
	pump              *ik.RecordPump
	watchers          map[string]*TailWatcher
	refreshTimer      *time.Ticker
	controlChan       chan struct {}
}

func (input *TailInput) Factory() ik.Plugin {
	return input.factory
}

func (input *TailInput) Port() ik.Port {
	return input.port
}

func (input *TailInput) Run() error {
	for {
		select {
		case <-input.refreshTimer.C:
			err := input.refreshWatchers()
			if err != nil {
				return err
			}
			return ik.Continue
		case <-input.controlChan:
			return nil
		}
	}
	return input.cleanup()
}

func (input *TailInput) cleanup() error {
	input.refreshTimer.Stop()
	errors := []error {
		input.pump.Shutdown(),
		input.positionFile.Dispose(),
	}
	err := (error)(nil)
	for _, err_ := range errors {
		input.logger.Println(err_.Error())
		if err == nil && err_ != nil {
			err = err_
		}
	}
	return err
}

func (input *TailInput) Shutdown() error {
	input.controlChan <- struct {}{}
	return nil
}

func (input *TailInput) Dispose() error {
	err := input.Shutdown()
	return err
}

func (input *TailInput) refreshWatchers() error {
	deleted := make([]*TailWatcher, 0, 8)
	added := make([]*TailWatcher, 0, 8)
	failed := true
	defer func () {
		if failed {
			for _, watcher := range added {
				watcher.Shutdown()
			}
		}
	}()
	newPaths, err := input.pathSet.Expand()
	if err != nil {
		return err
	}
	for path, watcher := range input.watchers {
		_, ok := newPaths[path]
		if !ok {
			deleted = append(deleted, watcher)
		}
	}
	for path, _ := range newPaths {
		_, ok := input.watchers[path]
		if !ok {
			watcher, err := input.openTailWatcher(path)
			if err != nil {
				return err
			}
			added = append(added, watcher)
		}
	}
	input.logger.Println("Refreshing watchers...")
	for _, watcher := range deleted {
		delete(input.watchers, watcher.tailFileInfo.Path())
		watcher.Shutdown()
		input.logger.Println(fmt.Sprintf("Deleted watcher for %s", watcher.tailFileInfo.Path()))
	}
	failed = false
	for _, watcher := range added {
		input.watchers[watcher.tailFileInfo.Path()] = watcher
		input.logger.Println(fmt.Sprintf("Added watcher for %s", watcher.tailFileInfo.Path()))
	}
	return nil
}

func newTailInput(
	factory *TailInputFactory,
	logger *log.Logger,
	engine ik.Engine,
	port ik.Port,
	pathSet *PathSet,
	lineParserFactory ik.LineParserFactory,
	tagPrefix string,
	tagSuffix string,
	rotateWait time.Duration,
	positionFilePath string,
	readFromHead bool,
	refreshInterval time.Duration,
	readBufferSize int,
) (*TailInput, error) {
	failed := true
	positionFile, err := openPositionFile(logger, positionFilePath)
	if err != nil {
		return nil, err
	}
	defer func () {
		if failed {
			positionFile.Dispose()
		}
	}()
	pump := ik.NewRecordPump(port, DefaultBacklogSize)
	defer func () {
		if failed {
			pump.Shutdown()
		}
	}()
	input := &TailInput {
		factory:          factory,
		engine:           engine,
		logger:           logger,
		port:             port,
		pathSet:          pathSet,
		tagPrefix:        tagPrefix,
		tagSuffix:        tagSuffix,
		rotateWait:	  rotateWait,
		readFromHead:     readFromHead,
		refreshInterval:  refreshInterval,
		readBufferSize:   readBufferSize,
		lineParserFactory:lineParserFactory,
		pump:             pump,
		positionFile:     positionFile,
		watchers:         make(map[string]*TailWatcher),
		controlChan:      make(chan struct {}, 1),
	}
	err = engine.Spawn(pump)
	if err != nil {
		return nil, err
	}
	err = input.refreshWatchers()
	if err != nil {
		return nil, err
	}
	failed = false
	input.refreshTimer = time.NewTicker(refreshInterval)
	return input, nil
}

func (factory *TailInputFactory) BindScorekeeper(*ik.Scorekeeper) {}

func (factory *TailInputFactory) Name() string {
	return "tail"
}

type PathSet struct {
	patterns []string
	fs http.FileSystem
}

func (pathSet *PathSet) Expand() (map[string]struct {}, error) {
	s := make(map[string]struct {})
	for _, pattern := range pathSet.patterns {
		paths, err := ik.Glob(pathSet.fs, pattern)
		if err != nil {
			return nil, err
		}
		for _, path := range paths {
			s[path] = struct {}{}
		}
	}
	return s, nil
}

func newPathSet(fs http.FileSystem, patterns []string) *PathSet {
	return &PathSet {
		patterns: patterns, 
		fs: fs,
	}
}

func splitAndStrip(s string) []string {
	comps := strings.Split(s, ",")
	retval := make([]string, len(comps))
	for i, c := range comps {
		retval[i] = strings.Trim(c, " \t")
	}
	return retval
}

func (factory *TailInputFactory) New(engine ik.Engine, config *ik.ConfigElement) (ik.Input, error) {
	tagPrefix := ""
	tagSuffix := ""
	rotateWait, _ := time.ParseDuration("5s")
	positionFilePath := ""
	readFromHead := false
	refreshInterval, _ := time.ParseDuration("1m")
	readBufferSize := 4096

	pathStr, ok := config.Attrs["path"]
	if !ok {
		return nil, errors.New("required attribute `path' is not specified")
	}
	pathSet := newPathSet(engine.Opener().FileSystem(), splitAndStrip(pathStr))
	tag, ok := config.Attrs["tag"]
	if !ok {
		return nil, errors.New("required attribute `tag' is not specified")
	}
	i := strings.IndexRune(tag, rune('*'))
	if i >= 0 {
		tagPrefix = tag[:i]
		tagSuffix = tag[i + 1:]
	} else {
		tagPrefix = tag
	}
	rotateWaitStr, ok := config.Attrs["rotate_wait"]
	if ok {
		var err error
		rotateWait, err = time.ParseDuration(rotateWaitStr)
		if err != nil {
			return nil, err
		}
	}
	positionFilePath, ok = config.Attrs["pos_file"]
	if !ok {
		return nil, errors.New("requires attribute `pos_file' is not specified")
	}
	readFromHeadStr, ok := config.Attrs["read_from_head"]
	if ok {
		var err error
		readFromHead, err = strconv.ParseBool(readFromHeadStr)
		if err != nil {
			return nil, err
		}
	}
	refreshIntervalStr, ok := config.Attrs["refresh_interval"]
	if ok {
		var err error
		refreshInterval, err = time.ParseDuration(refreshIntervalStr)
		if err != nil {
			return nil, err
		}
	}
	readBufferSizeStr, ok := config.Attrs["read_buffer_size"]
	if ok {
		var err error
		readBufferSize, err = strconv.Atoi(readBufferSizeStr)
		if err != nil {
			return nil, err
		}
	}

	format, ok := config.Attrs["format"]
	if !ok {
		return nil, errors.New("requires attribute `format' is not specified")
	}

	lineParserFactoryFactory := engine.LineParserPluginRegistry().LookupLineParserFactoryFactory(format)
	if lineParserFactoryFactory == nil {
		return nil, errors.New(fmt.Sprintf("Format `%s' is not supported", format))
	}
	lineParserFactory, err := lineParserFactoryFactory(engine, config)
	if err != nil {
		return nil, err
	}

	return newTailInput(
		factory,
		engine.Logger(),
		engine,
		engine.DefaultPort(),
		pathSet,
		lineParserFactory,
		tagPrefix,
		tagSuffix,
		rotateWait,
		positionFilePath,
		readFromHead,
		refreshInterval,
		readBufferSize,
	)
}

var _ = AddPlugin(&TailInputFactory{})
