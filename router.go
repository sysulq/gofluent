package main

import (
	"regexp"
	"sync/atomic"
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

		for re, outChan := range self.outChan {
			flag := re.MatchString(pack.Msg.Tag)
			if flag == true {
				atomic.AddInt32(&pack.RefCount, 1)
				outChan <- pack
			}
		}

		pack.Recycle()
	}
}
