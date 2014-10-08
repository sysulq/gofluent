package main

import (
	"fmt"
	"os"
)

type Input interface {
	New() interface{}
	Start(ctx chan Context) error
	Configure(f map[string]interface{}) error
}

var input_plugins = make(map[string]Input)

func RegisterInput(name string, input Input) {
	if input == nil {
		panic("input: Register input is nil")
	}

	if _, ok := input_plugins[name]; ok {
		panic("input: Register called twice for input " + name)
	}

	input_plugins[name] = input
}

func NewInput(ctx chan Context) {
	for _, input_config := range config.Inputs_config {
		f := input_config.(map[string]interface{})
		go func(f map[string]interface{}) {
			intput_type, ok := f["type"].(string)
			if !ok {
				fmt.Println("no type configured")
				os.Exit(-1)
			}

			input, ok := input_plugins[intput_type]
			if !ok {
				fmt.Println("unkown type ", intput_type)
				os.Exit(-1)
			}

			in := input.New()

			err := in.(Input).Configure(f)
			if err != nil {
				panic(err)
			}

			err = in.(Input).Start(ctx)
			if err != nil {
				panic(err)
			}
		}(f)
	}

	return
}
