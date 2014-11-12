package main

import ()

var output_plugins = make(map[string]func() interface{})

func RegisterOutput(name string, out func() interface{}) {
	if out == nil {
		Log("output: Register output is nil")
	}

	if _, ok := output_plugins[name]; ok {
		Log("output: Register called twice for output " + name)
	}

	output_plugins[name] = out
}
