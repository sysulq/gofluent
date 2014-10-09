package main

import (
	"github.com/ActiveState/tail"
	"os"
	"regexp"
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

	re := regexp.MustCompile(self.format)

	for line := range t.Lines {

		text := re.FindStringSubmatch(line.Text)
		for i, name := range re.SubexpNames() {
			if i != 0 {
				ctx <- Context{self.tag, map[string]string{name: text[i]}}
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
	RegisterInput("tail", &inputTail{})
}
