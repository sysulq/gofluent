package main

import (
	"fmt"
	"github.com/ActiveState/tail"
	"os"
	"regexp"
)

type InputTail struct {
}

func (inpuTail *InputTail) Start(source Source, ctx chan FluentdCtx) error {

	t, err := tail.TailFile(source.Path, tail.Config{Follow: true, Location: &tail.SeekInfo{0, os.SEEK_END}})
	if err != nil {
		return err
	}

	re := regexp.MustCompile(source.Format)

	for line := range t.Lines {

		text := re.FindStringSubmatch(line.Text)
		for i, name := range re.SubexpNames() {
			if i != 0 {
				fmt.Println(line.Text)
				ctx <- FluentdCtx{source, Match{}, map[string]string{name: text[i]}}
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
