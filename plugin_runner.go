package main

import (
	"log"
)

type InputRunner interface {
	InChan() chan *PipelinePack
	RouterChan() chan *PipelinePack
	Start(cf map[string]string)
}

type iRunner struct {
	inChan     chan *PipelinePack
	routerChan chan *PipelinePack
}

func NewInputRunner(in, router chan *PipelinePack) InputRunner {
	return &iRunner{
		inChan:     in,
		routerChan: router,
	}
}

func (this *iRunner) InChan() chan *PipelinePack {
	return this.inChan
}

func (this *iRunner) RouterChan() chan *PipelinePack {
	return this.routerChan
}

func (this *iRunner) Start(cf map[string]string) {
	intput_type, ok := cf["type"]
	if !ok {
		log.Fatalln("no type configured")
	}

	input, ok := input_plugins[intput_type]
	if !ok {
		log.Fatalln("unkown type ", intput_type)
	}

	in := input()

	err := in.(Input).Init(cf)
	if err != nil {
		log.Fatalln("in.(Input).Init", err)
	}

	err = in.(Input).Run(this)
	if err != nil {
		log.Fatalln("in.(Input).Run", err)
	}
}

type OutputRunner interface {
	InChan() chan *PipelinePack
	Start(cf map[string]string)
}

type oRunner struct {
	inChan chan *PipelinePack
}

func NewOutputRunner(in chan *PipelinePack) OutputRunner {
	return &oRunner{
		inChan: in,
	}
}

func (this *oRunner) InChan() chan *PipelinePack {
	return this.inChan
}

func (this *oRunner) Start(cf map[string]string) {
	output_type, ok := cf["type"]
	if !ok {
		log.Fatalln("no type configured")
	}

	output_plugin, ok := output_plugins[output_type]
	if !ok {
		log.Fatalln("unkown type ", output_type)
	}

	out := output_plugin()

	err := out.(Output).Init(cf)
	if err != nil {
		log.Fatalln("out.(Output).Init", err)
	}

	err = out.(Output).Run(this)
	if err != nil {
		log.Fatalln("out.(Output).Run", err)
	}
}
