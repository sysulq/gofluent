package main

import (
	"github.com/ActiveState/tail"
	"os"
	"time"
	"regexp"
	"fmt"
	"strings"
	"encoding/json"
)

type inputTail struct {
	path   string
	format string
	tag    string
}

func (self *inputTail) new() interface{} {
	return &inputTail{}
}

func (self *inputTail) configure(f map[string]interface{}) error {
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

	return nil
}

func (self *inputTail) start(ctx chan Context) error {

	t, err := tail.TailFile(self.path, tail.Config{Follow: true, Location: &tail.SeekInfo{0, os.SEEK_END}})
	if err != nil {
		return err
	}

	var re regexp.Regexp
	if string(self.format[0]) == string("/") || string(self.format[len(self.format)-1]) == string("/") {
		fmt.Println(strings.Trim(self.format, "/"))
		re = *regexp.MustCompile(self.format)
		self.format = "regexp"
	} else if self.format == "json" {

	}

	for line := range t.Lines {

		//text := re.FindSubmatch([]byte(line.Text))
		timeUnix := time.Now().Unix()
		data := make(map[string]string)

		if self.format == "regexp" {
			text := re.FindSubmatch([]byte(line.Text))
			if text == nil {
				continue
			}

			for i, name := range re.SubexpNames() {
				if i != 0 {
					fmt.Println(name, string(text[i]))
					data[name] = string(text[i])
				}
			}
		} else if self.format == "json" {
			err := json.Unmarshal([]byte(line.Text), &data)
			if err != nil {
				continue
			}
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
