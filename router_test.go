package main

import (
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
	println(a)
	if err != nil {
		t.Fail()
	}
	if a != "^a\\.(?:(?:b)|(?:c))\\.d$" {
		t.Fail()
	}
}
