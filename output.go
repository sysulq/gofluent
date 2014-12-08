package main

import (
	"log"
)

var output_plugins = make(map[string]func() interface{})

func RegisterOutput(name string, out func() interface{}) {
	if out == nil {
		log.Fatalln("output: Register output is nil")
	}

	if _, ok := output_plugins[name]; ok {
		log.Fatalln("output: Register called twice for output " + name)
	}

	output_plugins[name] = out
}
