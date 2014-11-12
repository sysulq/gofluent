package main

import (
	"flag"
	"os"
	"net"
	"bytes"
	"encoding/json"
	"reflect"
	"time"
	"log"
	"fmt"
	"math"
	"strconv"
	"sync/atomic"
	"github.com/moriyoshi/ik"
	"github.com/moriyoshi/ik/markup"
	"github.com/ugorji/go/codec"
	termutil "github.com/andrew-d/go-termutil"
)

type Record struct {
	Timestamp uint64
	Data map[string]interface{}
}

type IkBench struct {
	codec codec.MsgpackHandle
}

type IkBenchReportData struct {
	NumberOfRecordsSent int64
	LongestSubmissionTime time.Duration
	ShortestSubmissionTime time.Duration
	Now time.Time
	Start time.Time
}

type IkBenchReporter interface {
	ReportRecordsSent(data IkBenchReportData)
	ReportFinal(data IkBenchReportData)
}

type IkBenchParams struct {
	Host string
	Simple bool
	NumberOfRecordsToSubmit int
	NumberOfRecordsSentAtOnce int
	Concurrency int
	Tag string
	Data map[string]interface{}
	MaxRetryCount int
	ReportingFrequency int
	Reporter IkBenchReporter
}

func (ikb *IkBench) encodeEntrySingle(buf *bytes.Buffer, tag string, record Record) error {
	enc := codec.NewEncoder(buf, &ikb.codec)
	return enc.Encode([]interface{} { tag, record.Timestamp, record.Data })
}

func (ikb *IkBench) encodeEntryBulk(buf *bytes.Buffer, tag string, records []Record) error {
	enc := codec.NewEncoder(buf, &ikb.codec)
	return enc.Encode([]interface{} { tag, records })
}

func (ikb *IkBench) Submit(conn net.Conn, params *IkBenchParams) error {
	time_ := time.Now().Unix()
	records := make([]Record, params.NumberOfRecordsSentAtOnce)
	for i := 0; i < params.NumberOfRecordsSentAtOnce; i += 1 {
		records[i] = Record { Timestamp: uint64(time_), Data: params.Data }
	}
	buf := bytes.Buffer {}
	if params.Simple {
		for _, record := range records {
			err := ikb.encodeEntrySingle(&buf, params.Tag, record)
			if err != nil {
				return err
			}
		}
	} else {
		err := ikb.encodeEntryBulk(&buf, params.Tag, records)
		if err != nil {
			return err
		}
	}
	_, err := buf.WriteTo(conn)
	return err
}

func (ikb *IkBench) Run(logger *log.Logger, params *IkBenchParams) {
	numberOfRecordsSentAtOnce := params.NumberOfRecordsSentAtOnce
	numberOfAttempts := params.NumberOfRecordsToSubmit / numberOfRecordsSentAtOnce
	numberOfAttemptsPerProc := numberOfAttempts / params.Concurrency
	remainder := numberOfAttempts % params.Concurrency
	reportingFrequency := params.ReportingFrequency
	numberOfRecordsSent := int64(0)
	sync := make(chan int)
	start := time.Now()
	shortestSubmissionTime := time.Duration(-1)
	longestSubmissionTime := time.Duration(-1)
	var submissionStart time.Time
	for i := 0; i < params.Concurrency; i += 1 {
		r := 0
		if i < remainder {
			r = 1
		}
		go func(id int, attempts int) {
			retryCount := params.MaxRetryCount
			var conn net.Conn
			var err error
			defer func() {
				if conn != nil { conn.Close() }
			}()
			outer: for i := 0; i < attempts; i += 1 {
				for {
					if conn == nil {
						for {
							conn, err = net.Dial("tcp", params.Host)
							if err != nil {
								logger.Print(err.Error())
								retryCount -= 1
								if retryCount < 0 {
									logger.Fatal("retry count exceeded") // FIXME
								}
								continue
							}
							break
						}
					}
					if submissionStart.IsZero() {
						submissionStart = time.Now()
					}
					err = ikb.Submit(conn, params)
					if err != nil {
						err_, ok := err.(net.Error)
						if ok {
							if err_.Temporary() {
								continue
							}
							closeErr:= conn.Close()
							if closeErr != nil {
								logger.Print(closeErr.Error())
							}
							conn = nil
						}
						logger.Print(err.Error()) // FIXME
						break outer
					}
					now := time.Now()
					submissionTime := now.Sub(submissionStart)
					submissionStart = now
					if shortestSubmissionTime < 0 || shortestSubmissionTime > submissionTime {
						shortestSubmissionTime = submissionTime
					}
					if longestSubmissionTime < submissionTime {
						longestSubmissionTime = submissionTime
					}
					if atomic.AddInt64(&numberOfRecordsSent, int64(numberOfRecordsSentAtOnce)) % int64(reportingFrequency) == 0 {
						params.Reporter.ReportRecordsSent(IkBenchReportData {
							NumberOfRecordsSent: numberOfRecordsSent,
							ShortestSubmissionTime: shortestSubmissionTime,
							LongestSubmissionTime: longestSubmissionTime,
							Now: now,
							Start: start,
						})
					}
					break
				}
			}
			sync <- id
		}(i, numberOfAttemptsPerProc + r)
	}
	for i := 0; i < params.Concurrency; i += 1 {
		<-sync
	}
	params.Reporter.ReportFinal(IkBenchReportData {
		NumberOfRecordsSent: numberOfRecordsSent,
		ShortestSubmissionTime: shortestSubmissionTime,
		LongestSubmissionTime: longestSubmissionTime,
		Now: time.Now(),
		Start: start,
	})
}

func NewIkBench() *IkBench {
	codec_ := codec.MsgpackHandle {}
	codec_.MapType = reflect.TypeOf(map[string]interface{}(nil))
	codec_.RawToString = false
	codec_.StructToArray = true
	return &IkBench { codec: codec_ }
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s [-concurrent N] [-multi N] [-no-packed] [-host HOST] [-data JSON] tag count\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(255)
}

func exitWithMessage(message string, exitStatus int) {
	fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], message)
	os.Exit(exitStatus)
}

func exitWithError(err error, exitStatus int) {
	exitWithMessage(err.Error(), exitStatus)
}

type defaultReporter struct {
	renderer markup.MarkupRenderer
}

func (reporter *defaultReporter) ReportRecordsSent(data IkBenchReportData) {
	elapsed := float64(data.Now.Sub(data.Start)) / 1e9
	reporter.renderer.Render(&ik.Markup { []ik.MarkupChunk {
		ik.MarkupChunk {
			Attrs: 0,
			Text: fmt.Sprintf("%d records sent (%.3f seconds elapsed, %.3f records per second)\n", data.NumberOfRecordsSent, elapsed, float64(data.NumberOfRecordsSent) / elapsed),
		},
	} })
}

func (reporter *defaultReporter) ReportFinal(data IkBenchReportData) {
	elapsed := float64(data.Now.Sub(data.Start)) / 1e9
	reporter.renderer.Render(&ik.Markup { []ik.MarkupChunk {
		ik.MarkupChunk {
			Attrs: ik.Embolden | ik.Yellow,
			Text: "Benchmark Duration: ",
		},
		ik.MarkupChunk {
			Attrs: ik.Embolden,
			Text: fmt.Sprintf("%.3f seconds\n", elapsed),
		},
		ik.MarkupChunk {
			Attrs: ik.Embolden | ik.Yellow,
			Text: "Number of Records Submitted: ",
		},
		ik.MarkupChunk {
			Attrs: ik.Embolden,
			Text: fmt.Sprintf("%d\n", data.NumberOfRecordsSent),
		},
		ik.MarkupChunk {
			Attrs: ik.Embolden | ik.Yellow,
			Text: "Records per Second: ",
		},
		ik.MarkupChunk {
			Attrs: ik.Embolden,
			Text: fmt.Sprintf("%.3f\n", float64(data.NumberOfRecordsSent) / elapsed),
		},
		ik.MarkupChunk {
			Attrs: ik.Embolden | ik.Yellow,
			Text: "Average Submission Time: ",
		},
		ik.MarkupChunk {
			Attrs: ik.Embolden,
			Text: fmt.Sprintf("%.10f seconds\n", elapsed / float64(data.NumberOfRecordsSent)),
		},
		ik.MarkupChunk {
			Attrs: ik.Embolden,
			Text: "    Shortest: ",
		},
		ik.MarkupChunk {
			Attrs: ik.Embolden,
			Text: fmt.Sprintf("%.10f seconds\n", float64(data.ShortestSubmissionTime) / 1e9),
		},
		ik.MarkupChunk {
			Attrs: ik.Embolden,
			Text: "    Longest: ",
		},
		ik.MarkupChunk {
			Attrs: ik.Embolden,
			Text: fmt.Sprintf("%.10f seconds\n", float64(data.LongestSubmissionTime) / 1e9),
		},
	} })
}

func main() {
	var host string
	var simple bool
	var numberOfRecordsToSubmit int
	var numberOfRecordsSentAtOnce int
	var concurrency int
	var tag string
	var jsonString string
	flag.IntVar(&concurrency, "concurrent", 1, "number of goroutines")
	flag.IntVar(&numberOfRecordsSentAtOnce, "multi", 1, "send multiple records at once")
	flag.BoolVar(&simple, "no-packed", false, "don't use lazy deserialization optimize")
	flag.StringVar(&host, "host", "localhost:24224", "fluent host")
	flag.StringVar(&jsonString, "data", `{ "message": "test" }`, "data to send (in JSON)")
	flag.Parse()
	args := flag.Args()
	if len(args) < 2 {
		usage()
	}
	tag = args[0]
	numberOfRecordsToSubmit, err := strconv.Atoi(args[1])
	if err != nil {
		exitWithError(err, 255)
	}
	data := make(map[string]interface{})
	err = json.Unmarshal([]byte(jsonString), &data)
	if err != nil {
		exitWithError(err, 255)
	}
	if numberOfRecordsToSubmit % numberOfRecordsSentAtOnce != 0 {
		exitWithMessage("the value of 'count' must be a multiple of 'multi'", 255)
	}
	if numberOfRecordsToSubmit / numberOfRecordsSentAtOnce < concurrency {
		exitWithMessage("the value of 'concurrency' must be equal to or greater than the division of 'count' by 'multi'", 255)
	}
	var renderer markup.MarkupRenderer
	if termutil.Isatty(os.Stdout.Fd()) {
		renderer = &markup.TerminalEscapeRenderer { os.Stdout }
	} else {
		renderer = &markup.PlainRenderer { os.Stdout }
	}
	ikb := NewIkBench()
	ikb.Run(
		&log.Logger {},
		&IkBenchParams {
			Host: host,
			Simple: simple,
			NumberOfRecordsToSubmit: numberOfRecordsToSubmit,
			NumberOfRecordsSentAtOnce: numberOfRecordsSentAtOnce,
			Concurrency: concurrency,
			Tag: tag,
			Data: data,
			MaxRetryCount: 5,
			ReportingFrequency: int(math.Max(math.Pow(10, math.Ceil(math.Log10(float64(numberOfRecordsToSubmit))) - 1), 100)),
			Reporter: &defaultReporter { renderer: renderer },
		},
	)
}
