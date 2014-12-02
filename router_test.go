package main

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestBuildRegexpFromGlobPattern_0(t *testing.T) {
	a, err := BuildRegexpFromGlobPattern("a.b.c")
	if err != nil {
		t.Fail()
	}
	if a != "^a\\.b\\.c$" {
		t.Fail()
	}
}

func TestBuildRegexpFromGlobPattern_1(t *testing.T) {
	a, err := BuildRegexpFromGlobPattern("a.*.c")
	if err != nil {
		t.Fail()
	}
	if a != "^a\\.[^.]*\\.c$" {
		t.Fail()
	}
}

func TestBuildRegexpFromGlobPattern_2(t *testing.T) {
	a, err := BuildRegexpFromGlobPattern("**c")
	if err != nil {
		t.Fail()
	}
	if a != "^.*c$" {
		t.Fail()
	}
}

func TestBuildRegexpFromGlobPattern_3(t *testing.T) {
	a, err := BuildRegexpFromGlobPattern("**.c")
	if err != nil {
		t.Fail()
	}
	if a != "^(?:.*\\.|^)c$" {
		t.Fail()
	}
}

func TestBuildRegexpFromGlobPattern_4(t *testing.T) {
	a, err := BuildRegexpFromGlobPattern("a.{b,c}.d")

	if err != nil {
		t.Fail()
	}
	if a != "^a\\.(?:(?:b)|(?:c))\\.d$" {
		t.Fail()
	}
}

func TestRouter(t *testing.T) {
	in := make(chan *PipelinePack)
	out := make(chan *PipelinePack, 1)
	router := new(Router)

	router.Init()
	router.AddInChan(in)
	router.AddOutChan("test.**", out)
	go router.Loop()
	one := NewPipelinePack(in)
	one.Msg.Tag = "test.one"
	in <- one

	Convey("Confirm the outputs from outchan", t, func() {
		// Get the one pipelinepack and confirm it
		So(len(out), ShouldEqual, 1)
		res := <-out
		So(res.Msg.Tag, ShouldEqual, "test.one")

		// We should get the three pipelinepack,
		// instead of blocking the main loop of router
		two := NewPipelinePack(in)
		two.Msg.Tag = "test.two"
		in <- two

		three := NewPipelinePack(in)
		three.Msg.Tag = "test.three"
		in <- three

		So(len(out), ShouldEqual, 1)
		res = <-out
		So(res.Msg.Tag, ShouldEqual, "test.three")

	})
}
