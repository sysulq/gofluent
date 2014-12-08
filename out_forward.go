package main

import (
	"bytes"
	"github.com/ugorji/go/codec"
	"log"
	"net"
	"os"
	"reflect"
	"strconv"
	"time"
)

type OutputForward struct {
	host string
	port int

	connect_timeout int
	flush_interval  int

	codec   *codec.MsgpackHandle
	enc     *codec.Encoder
	conn    net.Conn
	buffer  bytes.Buffer
	backend BackendQueue
}

func (self *OutputForward) Init(config map[string]string) error {
	_codec := codec.MsgpackHandle{}
	_codec.MapType = reflect.TypeOf(map[string]interface{}(nil))
	_codec.RawToString = false
	_codec.StructToArray = true

	self.host = "localhost"
	self.port = 8888
	self.flush_interval = 10
	self.connect_timeout = 10
	self.codec = &_codec

	value := config["host"]
	if len(value) > 0 {
		self.host = value
	}

	value = config["port"]
	if len(value) > 0 {
		self.port, _ = strconv.Atoi(value)
	}

	value = config["connect_timeout"]
	if len(value) > 0 {
		self.connect_timeout, _ = strconv.Atoi(value)
	}

	value = config["flush_interval"]
	if len(value) > 0 {
		self.flush_interval, _ = strconv.Atoi(value)
	}

	return nil
}

func (self *OutputForward) Run(runner OutputRunner) error {
	l := log.New(os.Stderr, "", log.LstdFlags)
	self.backend = newDiskQueue("test", os.TempDir(), 1024768*100, 2500, 2*time.Second, l)

	tick := time.NewTicker(time.Second * time.Duration(self.flush_interval))

	for {
		select {
		case <-tick.C:
			{
				if self.backend.Depth() > 0 {
					log.Println("flush ", self.backend.Depth())
					self.flush()
				}
			}
		case pack := <-runner.InChan():
			{
				self.encodeRecordSet(pack.Msg)
				pack.Recycle()
			}
		}
	}

}

func (self *OutputForward) flush() error {
	if self.conn == nil {
		conn, err := net.DialTimeout("tcp", self.host+":"+strconv.Itoa(self.port), time.Second*time.Duration(self.connect_timeout))
		if err != nil {
			log.Println("%#v", err.Error())
			return err
		} else {
			self.conn = conn
		}
	}

	defer self.conn.Close()
	var buff bytes.Buffer
	for i := int64(0); i < self.backend.Depth(); i++ {
		buff.Read(<-self.backend.ReadChan())
	}

	n, err := buff.WriteTo(self.conn)
	if err != nil {
		log.Println("Write failed. size: %d, buf size: %d, error: %#v", n, buff.Len(), err.Error())
		self.conn = nil
		return err
	}
	if n > 0 {
		log.Printf("Forwarded: %d bytes (left: %d bytes)\n", n, buff.Len())
	}

	self.backend.Empty()
	self.conn = nil

	return nil

}

func (self *OutputForward) encodeRecordSet(msg Message) error {
	v := []interface{}{msg.Tag, msg.Timestamp, msg.Data}
	if self.enc == nil {
		self.enc = codec.NewEncoder(&self.buffer, self.codec)
	}
	err := self.enc.Encode(v)
	if err != nil {
		return err
	}
	self.backend.Put(self.buffer.Bytes())
	self.buffer.Reset()
	return err
}

func init() {
	RegisterOutput("forward", func() interface{} {
		return new(OutputForward)
	})
}
