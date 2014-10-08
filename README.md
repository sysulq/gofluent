gofluent
========

This program is something acting like fluentd rewritten in Go.

Introduction
========

Fluentd is originally written in CRuby, which has too many dependecies.
I hope fluentd to be simpler and cleaner, as its main feature is simplicity and rubostness.
So here I go, and need your help!

Installation
========

Follow steps below and have fun!
```
git clone https://github.com/hnlq715/gofluent
cd gofluent
export GOPATH=`pwd`
go get github.com/ActiveState/tail
go get github.com/t-k/fluent-logger-golang/fluent
go build
```
