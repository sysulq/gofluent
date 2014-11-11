package main

import ()

type Input interface {
	new() interface{}
	start(ctx chan Context) error
	configure(f map[string]string) error
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
