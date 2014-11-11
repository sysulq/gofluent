package main

import ()

type output interface {
	new() interface{}
	start(ctx chan Context) error
	configure(f map[string]string) error
}

var output_plugins = make(map[string]output)

func RegisterOutput(name string, out output) {
	if out == nil {
		Log("output: Register output is nil")
	}

	if _, ok := output_plugins[name]; ok {
		Log("output: Register called twice for output " + name)
	}

	output_plugins[name] = out
}
