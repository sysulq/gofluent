package ik

import (
	"errors"
	"testing"
)

type Foo struct {
	state string
	c     chan string
}

func (foo *Foo) Run() error {
	foo.state = "run"
	retval := <-foo.c
	foo.state = "stopped"
	return errors.New(retval)
}

func (foo *Foo) Shutdown() error {
	foo.c <- "ok"
	return nil
}

type Bar struct { c chan interface{} }

func (bar *Bar) Run() error {
	panic(<-bar.c)
}

func (bar *Bar) Shutdown() error {
	return nil
}

type Baz struct { message string }

func (baz *Baz) String() string {
	return baz.message
}

func TestSpawner_Spawn(t *testing.T) {
	spawner := NewSpawner()
	f := &Foo{"", make(chan string)}
	spawner.Spawn(f)
	err := spawner.GetStatus(f)
	if err != Continue {
		t.Fail()
	}
	f.c <- "result"
	spawner.Poll(f)
	err = spawner.GetStatus(f)
	if err == Continue {
		t.Fail()
	}
	if err.Error() != "result" {
		t.Fail()
	}
}

func TestSpawner_Panic1(t *testing.T) {
	spawner := NewSpawner()
	f := &Bar{make(chan interface{})}
	spawner.Spawn(f)
	err := spawner.GetStatus(f)
	if err != Continue {
		t.Fail()
	}
	f.c <- "PANIC"
	spawner.Poll(f)
	err = spawner.GetStatus(f)
	panicked, ok := err.(*Panicked)
	if !ok {
		t.Fail()
	}
	if panicked.Error() != "PANIC" {
		t.Fail()
	}
}

func TestSpawner_Panic2(t *testing.T) {
	spawner := NewSpawner()
	f := &Bar{make(chan interface{})}
	spawner.Spawn(f)
	err := spawner.GetStatus(f)
	if err != Continue {
		t.Fail()
	}
	f.c <- &Baz { "BAZ!" }
	spawner.Poll(f)
	err = spawner.GetStatus(f)
	panicked, ok := err.(*Panicked)
	if !ok {
		t.Fail()
	}
	if panicked.Error() != "(*Baz) BAZ!" {
		t.Fail()
	}
}
