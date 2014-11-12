package main

import (
	"fmt"
	"log"
	"os"
	"sync/atomic"
)

type Message struct {
	Tag       string
	Timestamp int64
	Data      map[string]string
}

type PipelinePack struct {
	MsgBytes    []byte
	Msg         Message
	RecycleChan chan *PipelinePack
	RefCount    int32
}

func NewPipelinePack(recycleChan chan *PipelinePack) (pack *PipelinePack) {
	msgBytes := make([]byte, 100)
	data := make(map[string]string)
	msg := Message{Data: data}
	return &PipelinePack{
		MsgBytes:    msgBytes,
		Msg:         msg,
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

type PipelineConfig struct {
	Gc                *GlobalConfig
	InputRunners      []interface{}
	OutputRunners     []interface{}
	router            Router
	inputRecycleChan  chan *PipelinePack
	outputRecycleChan chan *PipelinePack
}

func NewPipeLineConfig(gc *GlobalConfig) *PipelineConfig {
	config := new(PipelineConfig)
	config.router.Init()
	config.Gc = gc
	config.inputRecycleChan = make(chan *PipelinePack, gc.PoolSize)
	config.outputRecycleChan = make(chan *PipelinePack, gc.PoolSize)

	return config
}

func (this *PipelineConfig) LoadConfig(path string) error {
	configure, _ := ParseConfig(nil, path)
	for _, v := range configure.Root.Elems {
		if v.Name == "source" {
			this.InputRunners = append(this.InputRunners, v.Attrs)
		} else if v.Name == "match" {
			v.Attrs["tag"] = v.Args
			this.OutputRunners = append(this.OutputRunners, v.Attrs)
		}
	}

	return nil
}

func (this *PipelineConfig) InputRecycleChan() chan *PipelinePack {
	return this.inputRecycleChan
}

func (this *PipelineConfig) OutputRecycleChan() chan *PipelinePack {
	return this.outputRecycleChan
}

func Run(config *PipelineConfig) {
	log.Println("Starting gofluent...")

	for i := 0; i < config.Gc.PoolSize; i++ {
		iPack := NewPipelinePack(config.InputRecycleChan())
		config.InputRecycleChan() <- iPack
	}

	rChan := make(chan *PipelinePack, config.Gc.PoolSize)
	iRunner := NewInputRunner(config.InputRecycleChan(), rChan)

	config.router.AddInChan(iRunner.RouterChan())

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

			err = in.(Input).Run(iRunner)
			if err != nil {
				fmt.Println(err)
				os.Exit(-1)
			}
		}(f)
	}

	for _, output_config := range config.OutputRunners {
		f := output_config.(map[string]string)
		inChan := make(chan *PipelinePack, config.Gc.PoolSize)
		for i := 0; i < config.Gc.PoolSize; i++ {
			oPack := NewPipelinePack(inChan)
			config.OutputRecycleChan() <- oPack
		}
		oRunner := NewOutputRunner(inChan)
		config.router.AddOutChan(f["tag"], oRunner.InChan())

		go func(f map[string]string, oRunner OutputRunner) {
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

			err = out.(Output).Run(oRunner)
			if err != nil {
				Log(err)
			}
		}(f, oRunner)
	}

	config.router.Loop()
}
