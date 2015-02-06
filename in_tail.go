package main

import (
	"github.com/ActiveState/tail"
	"github.com/ugorji/go/codec"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type inputTail struct {
	path     string
	format   string
	tag      string
	pos_file string

	offset        int64
	sync_interval int
	codec         *codec.JsonHandle
	time_key      string
}

func (self *inputTail) Init(f map[string]string) error {

	self.sync_interval = 2

	value := f["path"]
	if len(value) > 0 {
		self.path = value
	}

	value = f["format"]
	if len(value) > 0 {
		self.format = value
		if value == "json" {
			_codec := codec.JsonHandle{}
			_codec.MapType = reflect.TypeOf(map[string]interface{}(nil))
			self.codec = &_codec

			value = f["time_key"]
			if len(value) > 0 {
				self.time_key = value
			} else {
				self.time_key = "time"
			}
		}
	}

	value = f["tag"]
	if len(value) > 0 {
		self.tag = value
	}

	value = f["pos_file"]
	if len(value) > 0 {
		self.pos_file = value

		str, err := ioutil.ReadFile(self.pos_file)
		if err != nil {
			log.Println("ioutil.ReadFile:", err)
		}

		f, err := os.Open(self.path)
		if err != nil {
			log.Println("os.Open:", err)
		}

		info, err := f.Stat()
		if err != nil {
			log.Println("f.Stat:", err)
			self.offset = 0
		} else {
			offset, _ := strconv.Atoi(string(str))
			if int64(offset) > info.Size() {
				self.offset = info.Size()
			} else {
				self.offset = int64(offset)
			}
		}
	}

	value = f["sync_interval"]
	if len(value) > 0 {
		sync_interval, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		self.sync_interval = sync_interval
	}

	return nil
}

func (self *inputTail) Run(runner InputRunner) error {
	defer func() {
		if err := recover(); err != nil {
			logs.Fatalln("recover panic at err:", err)
		}
	}()

	var seek int
	if self.offset > 0 {
		seek = os.SEEK_SET
	} else {
		seek = os.SEEK_END
	}

	t, err := tail.TailFile(self.path, tail.Config{
		Poll:      true,
		ReOpen:    true,
		Follow:    true,
		MustExist: false,
		Location:  &tail.SeekInfo{int64(self.offset), seek}})
	if err != nil {
		return err
	}

	f, err := os.OpenFile(self.pos_file, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatalln("os.OpenFile", err)
	}
	defer f.Close()

	var re regexp.Regexp
	if string(self.format[0]) == string("/") || string(self.format[len(self.format)-1]) == string("/") {
		format := strings.Trim(self.format, "/")
		trueformat := regexp.MustCompile("\\(\\?<").ReplaceAllString(format, "(?P<")
		if trueformat != format {
			log.Printf("pos_file:%s, format:%s", self.path, trueformat)
		}
		re = *regexp.MustCompile(trueformat)
		self.format = "regexp"
	} else if self.format == "json" {

	}

	tick := time.NewTicker(time.Second * time.Duration(self.sync_interval))
	count := 0

	for {
		select {
		case <-tick.C:
			{
				if count > 0 {
					offset, err := t.Tell()
					if err != nil {
						log.Println("Tell return error: ", err)
						continue
					}

					str := strconv.Itoa(int(offset))

					_, err = f.WriteAt([]byte(str), 0)
					if err != nil {
						log.Println("f.WriteAt", err)
						return err
					}

					count = 0
				}
			}
		case line := <-t.Lines:
			{
				pack := <-runner.InChan()

				pack.MsgBytes = []byte(line.Text)
				pack.Msg.Tag = self.tag
				pack.Msg.Timestamp = line.Time.Unix()

				if self.format == "regexp" {
					text := re.FindSubmatch([]byte(line.Text))
					if text == nil {
						pack.Recycle()
						continue
					}

					for i, name := range re.SubexpNames() {
						if len(name) > 0 {
							pack.Msg.Data[name] = string(text[i])
						}
					}
				} else if self.format == "json" {
					dec := codec.NewDecoderBytes([]byte(line.Text), self.codec)
					err := dec.Decode(&pack.Msg.Data)
					if err != nil {
						log.Println("json.Unmarshal", err)
						pack.Recycle()
						continue
					} else {
						t, ok := pack.Msg.Data[self.time_key]
						if ok {
							if time, xx := t.(uint64); xx {
								pack.Msg.Timestamp = int64(time)
								delete(pack.Msg.Data, self.time_key)
							} else if time64, oo := t.(int64); oo {
								pack.Msg.Timestamp = time64
								delete(pack.Msg.Data, self.time_key)
							} else {
								log.Println("time is not int64, ", t, " typeof:", reflect.TypeOf(t))
								pack.Recycle()
								continue
							}
						}
					}
				}

				count++
				runner.RouterChan() <- pack
			}
		}
	}

	err = t.Wait()
	if err != nil {
		return err
	}

	return err
}

func init() {
	RegisterInput("tail", func() interface{} {
		return new(inputTail)
	})
}
