package main

import (
	"log"
	"sync/atomic"
)

type Message struct {
	Tag       string
	Timestamp int64
	Data      map[string]interface{}
}

type PipelinePack struct {
	MsgBytes    []byte
	Msg         Message
	RecycleChan chan *PipelinePack
	RefCount    int32
}

func NewPipelinePack(recycleChan chan *PipelinePack) (pack *PipelinePack) {
	msgBytes := make([]byte, 100)
	data := make(map[string]interface{})
	msg := Message{Data: data}
	return &PipelinePack{
		MsgBytes:    msgBytes,
		Msg:         msg,
		RecycleChan: recycleChan,
		RefCount:    1,
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
	Gc            *GlobalConfig
	InputRunners  []interface{}
	OutputRunners []interface{}
	router        Router
}

func NewPipeLineConfig(gc *GlobalConfig) *PipelineConfig {
	config := new(PipelineConfig)
	config.router.Init()
	config.Gc = gc

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

func Run(config *PipelineConfig) {
	log.Println("Starting gofluent...")

	rChan := make(chan *PipelinePack, config.Gc.PoolSize)
	config.router.AddInChan(rChan)

	for _, input_config := range config.InputRunners {
		cf := input_config.(map[string]string)

		InputRecycleChan := make(chan *PipelinePack, config.Gc.PoolSize)
		for i := 0; i < config.Gc.PoolSize; i++ {
			iPack := NewPipelinePack(InputRecycleChan)
			InputRecycleChan <- iPack
		}
		iRunner := NewInputRunner(InputRecycleChan, rChan)

		go iRunner.Start(cf)
	}

	for _, output_config := range config.OutputRunners {
		cf := output_config.(map[string]string)

		inChan := make(chan *PipelinePack, config.Gc.PoolSize)
		oRunner := NewOutputRunner(inChan)
		config.router.AddOutChan(cf["tag"], oRunner.InChan())

		go oRunner.Start(cf)
	}

	config.router.Loop()
}
