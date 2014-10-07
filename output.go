package main

import (
	"os"
	"fmt"
)

type Output interface {
	New() interface{}
	Start(ctx chan Context) error
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

func NewOutput(ctx chan Context) error {
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

			out := output.New()
			fmt.Println("out:",out)
			err := out.(Output).Configure(f)
			if err != nil {
				panic(err)
			}
			fmt.Println(out)

			err = out.(Output).Start(ctx)
			if err != nil {
				panic(err)
			}
		}(f)
	}

	return nil
}

