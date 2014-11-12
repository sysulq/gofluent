package main

import ()

var input_plugins = make(map[string]func() interface{})

func RegisterInput(name string, input func() interface{}) {
	if input == nil {
		panic("input: Register input is nil")
	}

	if _, ok := input_plugins[name]; ok {
		panic("input: Register called twice for input " + name)
	}

	input_plugins[name] = input
}
