package main

import (
	"fmt"
)

type OutputStdout struct {
}

func (self *OutputStdout) new() interface{} {
	return &OutputStdout{}
}

func (self *OutputStdout) configure(f map[string]interface{}) error {
	return nil
}

func (self *OutputStdout) start(ctx chan Context) error {
	go func(ctx chan Context) {
		for {
			ch := <-ctx
			fmt.Println(ch.data)
		}
	}(ctx)

	return nil
}

func init() {
	RegisterOutput("stdout", &OutputStdout{})
}
