package main

import (
	. "github.com/smartystreets/goconvey/convey"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"
)

func TestTailJson(t *testing.T) {
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

func TestTailRegex(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	cf := make(map[string]string)
	cf["path"] = "/tmp/test.file"
	cf["format"] = "/^(?P<data>.*), (?P<hello>.*)$/"
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
	f.Write([]byte("test, world\n"))
	f.Close()

	pack := <-iRunner.RouterChan()

	Convey("Create tail file and write something", t, func() {
		Convey("pipelinepack should get the regex map", func() {
			So(pack.Msg.Tag, ShouldEqual, "test")
			So(pack.Msg.Data["data"], ShouldEqual, "test")
			So(pack.Msg.Data["hello"], ShouldEqual, "world")
		})
	})

	os.Remove(cf["pos_file"])
	os.Remove(cf["path"])
}

func TestSyncIntervalBug(t *testing.T) {
	cf := make(map[string]string)
	cf["path"] = "/tmp/test.file"
	cf["format"] = "/^(?P<data>.*), (?P<hello>.*)$/"
	cf["tag"] = "test"
	cf["sync_interval"] = "4"
	cf["pos_file"] = "./test.pos"
	tail := new(inputTail)
	err := tail.Init(cf)
	if err != nil {
		t.Fail()
	}

	rChan := make(chan *PipelinePack)
	InputRecycleChan := make(chan *PipelinePack, 1)
	iPack := NewPipelinePack(InputRecycleChan)
	InputRecycleChan <- iPack
	iRunner := NewInputRunner(InputRecycleChan, rChan)

	f, _ := os.OpenFile(cf["path"], os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)

	go tail.Run(iRunner)
	time.Sleep(1 * time.Second)
	f.Write([]byte("test, world\n"))
	f.Close()

	os.Remove(cf["pos_file"])
	os.Remove(cf["path"])

}

func TestNamedRegexMatch(t *testing.T) {
	cf := make(map[string]string)
	cf["path"] = "/tmp/test.file"
	cf["format"] = "/(^(?P<data>.*)), (?P<hello>.*)$/"
	cf["tag"] = "test"
	cf["sync_interval"] = "4"
	cf["pos_file"] = "./test.pos"
	tail := new(inputTail)
	err := tail.Init(cf)
	if err != nil {
		t.Fail()
	}

	rChan := make(chan *PipelinePack)
	InputRecycleChan := make(chan *PipelinePack, 1)
	iPack := NewPipelinePack(InputRecycleChan)
	InputRecycleChan <- iPack
	iRunner := NewInputRunner(InputRecycleChan, rChan)

	f, _ := os.OpenFile(cf["path"], os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)

	go tail.Run(iRunner)
	time.Sleep(1 * time.Second)
	f.Write([]byte("test, world\n"))
	f.Close()

	pack := <-iRunner.RouterChan()

	if len(pack.Msg.Data) == 3 {
		t.Fatal(pack.Msg.Data)
	}

	os.Remove(cf["pos_file"])
	os.Remove(cf["path"])

}

func TestPcreStyleRegex(t *testing.T) {
	cf := make(map[string]string)
	cf["path"] = "/tmp/test.file"
	cf["format"] = "/(^(?<data>.*)), (?<hello>.*)$/"
	cf["tag"] = "test"
	cf["sync_interval"] = "4"
	cf["pos_file"] = "./test.pos"
	tail := new(inputTail)
	err := tail.Init(cf)
	if err != nil {
		t.Fail()
	}

	rChan := make(chan *PipelinePack)
	InputRecycleChan := make(chan *PipelinePack, 1)
	iPack := NewPipelinePack(InputRecycleChan)
	InputRecycleChan <- iPack
	iRunner := NewInputRunner(InputRecycleChan, rChan)

	f, _ := os.OpenFile(cf["path"], os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)

	go tail.Run(iRunner)
	time.Sleep(1 * time.Second)
	f.Write([]byte("test, world\n"))
	f.Close()

	pack := <-iRunner.RouterChan()

	if len(pack.Msg.Data) == 3 {
		t.Fatal(pack.Msg.Data)
	}

	os.Remove(cf["pos_file"])
	os.Remove(cf["path"])
}
