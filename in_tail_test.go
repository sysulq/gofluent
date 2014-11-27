package main

import (
	. "github.com/smartystreets/goconvey/convey"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"
)

func Test_json(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	cf := make(map[string]string)
	cf["path"] = "/tmp/test.file"
	cf["format"] = "json"
	cf["tag"] = "test"
	cf["pos_file"] = "/tmp/test.pos"
	tail := new(inputTail)

	Convey("Init tail plugin", t, func() {
		err := tail.Init(cf)
		Convey("tail.Init err should be nil", func() {
			So(err, ShouldEqual, nil)
		})
	})

	rChan := make(chan *PipelinePack)
	InputRecycleChan := make(chan *PipelinePack, 1)
	iPack := NewPipelinePack(InputRecycleChan)
	InputRecycleChan <- iPack
	iRunner := NewInputRunner(InputRecycleChan, rChan)
	os.Remove(cf["pos_file"])
	os.Remove(cf["path"])

	f, _ := os.OpenFile(cf["path"], os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)

	go tail.Run(iRunner)
	time.Sleep(1 * time.Second)
	f.Write([]byte("{\"data\":\"test\",\"hello\":\"world\"}\n"))
	f.Close()

	pack := <-iRunner.RouterChan()

	Convey("Create tail file and write something", t, func() {
		Convey("pipelinepack should get the json map", func() {
			So(pack.Msg.Data["data"], ShouldEqual, "test")
			So(pack.Msg.Data["hello"], ShouldEqual, "world")
		})
	})

	os.Remove(cf["pos_file"])
	os.Remove(cf["path"])
}
