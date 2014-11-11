package main

type InputRunner interface {
	InChan() chan *PipelinePack
	RouterChan() chan *PipelinePack
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

type OutputRunner interface {
	InChan() chan *PipelinePack
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
