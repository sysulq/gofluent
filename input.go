package main

import (
	"os"
	"fmt"
)

type Input interface {
	Start(ctx chan FluentdCtx) error
	Configure(f map[string]interface{}) error
}

var inputs = make(map[string]Input)

func RegisterInput(name string, input Input) {
	if input == nil {
		panic("input: Register input is nil")
	}

	if _, ok := inputs[name]; ok {
		panic("input: Register called twice for input " + name)
	}

	inputs[name] = input
}

func NewInput(ctx chan FluentdCtx) {
	for _, input_config := range config.Inputs_config {
		f := input_config.(map[string]interface{})
		go func(f map[string]interface{}) {
			intput_type, ok := f["type"].(string)
			if !ok {
				fmt.Println("no type configured")
				os.Exit(-1)
			}

			input, ok := inputs[intput_type]
			if !ok {
				fmt.Println("unkown type ", intput_type)
				os.Exit(-1)
			}

			err := input.Configure(f)
			if err != nil {
				panic(err)
			}

			err = input.Start(ctx)
			if err != nil {
				panic(err)
			}
		}(f)
	}

	return
}
