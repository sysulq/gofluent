package main

import (
	"fmt"
	"log"
	"os"
	"sync/atomic"
)

type PipelinePack struct {
	MsgBytes    []byte
	Ctx         Context
	RecycleChan chan *PipelinePack
	RefCount    int32
}

func NewPipelinePack(recycleChan chan *PipelinePack) (pack *PipelinePack) {
	msgBytes := make([]byte, 100)
	return &PipelinePack{
		MsgBytes:    msgBytes,
		RecycleChan: recycleChan,
		RefCount:    int32(1),
	}
}

func (this *PipelinePack) Zero() {
	this.MsgBytes = this.MsgBytes[:cap(this.MsgBytes)]
	this.RefCount = 1
}

func (this *PipelinePack) Recycle() {
	cnt := atomic.AddInt32(&this.RefCount, -1)
	if cnt == 0 {
		this.Zero()
		this.RecycleChan <- this
	}
}

func Run(config *PipelineConfig) {
	log.Println("Starting gofluent...")
	ctxInput := make(chan Context, 10)
	config.router.AddInChan(ctxInput)

	for _, input_config := range config.InputRunners {
		f := input_config.(map[string]string)
		go func(f map[string]string) {
			intput_type, ok := f["type"]
			if !ok {
				fmt.Println("no type configured")
				os.Exit(-1)
			}

			input, ok := input_plugins[intput_type]
			if !ok {
				fmt.Println("unkown type ", intput_type)
				os.Exit(-1)
			}

			in := input()

			err := in.(Input).Init(f)
			if err != nil {
				fmt.Println(err)
				os.Exit(-1)
			}

			err = in.(Input).Run(ctxInput)
			if err != nil {
				fmt.Println(err)
				os.Exit(-1)
			}
		}(f)
	}

	for _, output_config := range config.OutputRunners {
		f := output_config.(map[string]string)
		tmpch := make(chan Context)
		tag := f["tag"]
		config.router.AddOutChan(tag, tmpch)
		go func(f map[string]string, tmpch chan Context) {
			output_type, ok := f["type"]
			if !ok {
				fmt.Println("no type configured")
				os.Exit(-1)
			}

			output_plugin, ok := output_plugins[output_type]
			if !ok {
				Log("unkown type ", output_type)
				os.Exit(-1)
			}

			out := output_plugin()

			err := out.(Output).Init(f)
			if err != nil {
				Log(err)
			}

			err = out.(Output).Run(tmpch)
			if err != nil {
				Log(err)
			}
		}(f, tmpch)
	}

	config.router.Loop()
}
