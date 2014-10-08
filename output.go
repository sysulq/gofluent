package main

import (
	"fmt"
	"os"
	"regexp"
)

type Output interface {
	New() interface{}
	Start(ctx chan Context) error
	Configure(f map[string]interface{}) error
}

var output_plugins = make(map[string]Output)

func RegisterOutput(name string, output Output) {
	if output == nil {
		panic("output: Register output is nil")
	}

	if _, ok := output_plugins[name]; ok {
		panic("output: Register called twice for output " + name)
	}

	output_plugins[name] = output
}

func NewOutput(ctx chan Context) error {
	outChan := make([]chan Context, 0)
	for _, output_config := range config.Outputs_config {
		f := output_config.(map[string]interface{})
		tmpch := make(chan Context)
		outChan = append(outChan, tmpch)
		go func(f map[string]interface{}, tmpch chan Context) {
			output_type, ok := f["type"].(string)
			if !ok {
				fmt.Println("no type configured")
				os.Exit(-1)
			}

			output, ok := output_plugins[output_type]
			if !ok {
				fmt.Println("unkown type ", output_type)
				os.Exit(-1)
			}

			out := output.New()
			err := out.(Output).Configure(f)
			if err != nil {
				panic(err)
			}

			err = out.(Output).Start(tmpch)
			if err != nil {
				panic(err)
			}
		}(f, tmpch)
	}

	for {
		s := <-ctx
		for i, o := range outChan {
			f := config.Outputs_config[i].(map[string]interface{})
			tag := f["tag"].(string)

			tagre := regexp.MustCompile(tag)

			flag := tagre.MatchString(s.tag)
			if flag == false {
				continue
			}

			o <- s
		}
	}

	return nil
}
