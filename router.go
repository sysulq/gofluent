package main

import (
	"regexp"
)

type Router struct {
	inChan  chan Context
	outChan map[*regexp.Regexp]chan Context
}

func (self *Router) Init() {
	self.outChan = make(map[*regexp.Regexp]chan Context)
}

func (self *Router) AddOutChan(matchtag string, outChan chan Context) error {
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

func (self *Router) AddInChan(inChan chan Context) {
	self.inChan = inChan
}

func (self *Router) Loop() {
	for {
		ctx := <-self.inChan
		for k, v := range self.outChan {
			flag := k.MatchString(ctx.tag)
			if flag == true {
				v <- ctx
			}
		}
	}
}
