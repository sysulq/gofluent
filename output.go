package main

import (
	"os"
	"fmt"
)

type Output interface {
	Start(ctx chan FluentdCtx) error
	Configure(f map[string]interface{}) error
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
	for _, output_config := range config.Outputs_config {
		f := output_config.(map[string]interface{})
		go func(f map[string]interface{}) {
			output_type, ok := f["type"].(string)
			if !ok {
				fmt.Println("no type configured")
				os.Exit(-1)
			}

			output, ok := outputs[output_type]
			if !ok {
				fmt.Println("unkown type ", output_type)
				os.Exit(-1)				
			}

			err := output.Configure(f)
			if err != nil {
				panic(err)
			}

			err = output.Start(ctx)
			if err != nil {
				panic(err)
			}
		}(f)
	}

	return nil
}

