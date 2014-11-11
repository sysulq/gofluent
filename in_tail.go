package main

import (
	"encoding/json"
	"github.com/ActiveState/tail"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type inputTail struct {
	path     string
	format   string
	tag      string
	pos_file string

	offset int64
}

func (self *inputTail) new() interface{} {
	return &inputTail{offset: 0}
}

func (self *inputTail) configure(f map[string]string) error {
	var value interface{}

	value = f["path"]
	if value != nil {
		self.path = value.(string)
	}

	value = f["format"]
	if value != nil {
		self.format = value.(string)
	}

	value = f["tag"]
	if value != nil {
		self.tag = value.(string)
	}

	value = f["pos_file"]
	if value != nil {
		self.pos_file = value.(string)

		str, err := ioutil.ReadFile(self.pos_file)
		if err != nil {
			Log(err)
		}

		f, err := os.Open(self.path)
		if err != nil {
			Log(err)
		}

		info, err := f.Stat()
		if err != nil {
			Log(err)
		} else {
			offset, _ := strconv.Atoi(string(str))
			if int64(offset) > info.Size() {
				self.offset = info.Size()
			} else {
				self.offset = int64(offset)
			}
		}
	}

	return nil
}

func (self *inputTail) start(ctx chan Context) error {

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

	var re regexp.Regexp
	if string(self.format[0]) == string("/") || string(self.format[len(self.format)-1]) == string("/") {
		re = *regexp.MustCompile(strings.Trim(self.format, "/"))
		self.format = "regexp"
	} else if self.format == "json" {

	}

	f, err := os.OpenFile(self.pos_file, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		Log(err)
	}
	defer f.Close()

	for line := range t.Lines {

		timeUnix := line.Time.Unix()
		data := make(map[string]string)

		if self.format == "regexp" {
			text := re.FindSubmatch([]byte(line.Text))
			if text == nil {
				continue
			}

			for i, name := range re.SubexpNames() {
				if i != 0 {
					data[name] = string(text[i])
				}
			}
		} else if self.format == "json" {
			err := json.Unmarshal([]byte(line.Text), &data)
			if err != nil {
				continue
			}
		}

		offset, err := t.Tell()
		if err != nil {
			Log("Tell return error: ", err)
		}

		str := strconv.Itoa(int(offset))

		_, err = f.WriteString(str)
		if err != nil {
			Log(err)
		}

		record := Record{timeUnix, data}
		ctx <- Context{self.tag, record}
	}

	err = t.Wait()
	if err != nil {
		return err
	}

	return err
}

func init() {
	RegisterInput("tail", &inputTail{})
}
