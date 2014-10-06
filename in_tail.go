package main

import (
	"fmt"
	"github.com/ActiveState/tail"
	"os"
	"regexp"
)

type InputTail struct {
	path   string
	format string
	tag string
}

func (self *InputTail) Configure(f map[string]interface{}) error {
	self.path = f["path"].(string)
	self.format = f["format"].(string)
	self.tag = f["tag"].(string)

	return nil
}

func (self *InputTail) Start(ctx chan FluentdCtx) error {

	t, err := tail.TailFile(self.path, tail.Config{Follow: true, Location: &tail.SeekInfo{0, os.SEEK_END}})
	if err != nil {
		return err
	}

	re := regexp.MustCompile(self.format)

	for line := range t.Lines {

		text := re.FindStringSubmatch(line.Text)
		for i, name := range re.SubexpNames() {
			if i != 0 {
				fmt.Println(line.Text)
				ctx <- FluentdCtx{self.tag, map[string]string{name: text[i]}}
			}
		}
	}

	err = t.Wait()
	if err != nil {
		return err
	}

	return err
}

func NewInputTail() *InputTail {
	return &InputTail{}
}

func init() {
	RegisterInput("tail", NewInputTail())
}
