package main

import (
	"fmt"
)

type Output interface {
	Start(ctx chan FluentdCtx) error
	Configure(i int, filePath string) error
}

var outputs = make(map[string]Output)

func RegisterOutput(name string, output Output) {
	if output == nil {
		panic("output: Register output is nil")
	}

	if _, ok := outputs[name]; ok {
		panic("output: Register called twice for output " + name)
	}

	outputs[name] = output
}

func NewOutput(ctx chan FluentdCtx) error {
	for i := range config.Matches {
		go func(i int) {
			output, ok := outputs[config.Matches[i].Type]
			if !ok {
				fmt.Println("unkown type", config.Matches[i].Type)
			}

			err := output.Configure(i, "config.json")
			if err != nil {
				output = nil
			}

			err = output.Start(ctx)
			if err != nil {
				output = nil
			}
		}(i)
	}

	return nil
}

