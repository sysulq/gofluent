package main

import (
	"fmt"
)

type OutputStdout struct {
}

func (self *OutputStdout) New() interface{} {
	return &OutputStdout{}
}

func (self *OutputStdout) Configure(f map[string]interface{}) error {
	return nil
}

func (self *OutputStdout) Start(ctx chan Context) error {
	go func(ctx chan Context) {
		for {
			ch := <-ctx
			fmt.Println(ch.data)
		}
	}(ctx)

	return nil
}

func NewOutputStdout() *OutputStdout {
	return &OutputStdout{}
}

func init() {
	RegisterOutput("stdout", NewOutputStdout())
}
