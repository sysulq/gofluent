package main

import (
	"fmt"
)

type Input interface {
	Start(source Source, ctx chan FluentdCtx) error
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
	for i := range config.Sources {
		go func(i int) {
			input, ok := inputs[config.Sources[i].Type]
			if !ok {
				fmt.Println("unkown type", config.Sources[i].Type)
			}

			err := input.Start(config.Sources[i], ctx)
			if err != nil {
				input = nil
			}
		}(i)
	}

	return
}
