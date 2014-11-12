package main

import (
	"regexp"
)

type Router struct {
	inChan  chan *PipelinePack
	outChan map[*regexp.Regexp]chan *PipelinePack
}

func (self *Router) Init() {
	self.outChan = make(map[*regexp.Regexp]chan *PipelinePack)
}

func (self *Router) AddOutChan(matchtag string, outChan chan *PipelinePack) error {
	chunk, err := BuildRegexpFromGlobPattern(matchtag)
	if err != nil {
		return err
	}

	re, err := regexp.Compile(chunk)
	if err != nil {
		return err
	}

	self.outChan[re] = outChan
	return nil
}

func (self *Router) AddInChan(inChan chan *PipelinePack) {
	self.inChan = inChan
}

func (self *Router) Loop() {
	for {
		pack := <-self.inChan
		for k, v := range self.outChan {
			flag := k.MatchString(pack.Msg.Tag)
			if flag == true {
				v <- pack
			}
		}
	}
}
