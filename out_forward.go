package main

import (
	"bytes"
	"github.com/ugorji/go/codec"
	"log"
	"net"
	"reflect"
	"strconv"
	"time"
)

type OutputForward struct {
	host string
	port int

	connect_timeout int
	flush_interval  int

	logger *log.Logger
	codec  *codec.MsgpackHandle
	enc    *codec.Encoder
	conn   net.Conn
	buffer bytes.Buffer
}

func (self *OutputForward) Init(f map[string]string) error {
	_codec := codec.MsgpackHandle{}
	_codec.MapType = reflect.TypeOf(map[string]interface{}(nil))
	_codec.RawToString = false
	_codec.StructToArray = true

	self.host = "localhost"
	self.port = 8888
	self.flush_interval = 10
	self.connect_timeout = 10
	self.codec = &_codec

	var value interface{}

	value = f["host"]
	if value != nil {
		self.host = value.(string)
	}

	value = f["port"]
	if value != nil {
		self.port = int(value.(float64))
	}

	value = f["connect_timeout"]
	if value != nil {
		self.connect_timeout = int(value.(float64))
	}

	value = f["flush_interval"]
	if value != nil {
		self.flush_interval = int(value.(float64))
	}

	return nil
}

func (self *OutputForward) Run(runner OutputRunner) error {

	tick := time.NewTicker(time.Second * time.Duration(self.flush_interval))

	for {
		select {
		case <-tick.C:
			{
				if self.buffer.Len() > 0 {
					Log("flush ", self.buffer.Len())
					self.flush()
				}
			}
		case pack := <-runner.InChan():
			{
				self.encodeRecordSet(pack.Ctx)
				pack.Recycle()
			}
		}
	}

}

func (self *OutputForward) flush() error {
	if self.conn == nil {
		conn, err := net.DialTimeout("tcp", self.host+":"+strconv.Itoa(self.port), time.Second*time.Duration(self.connect_timeout))
		if err != nil {
			Log("%#v", err.Error())
			return err
		} else {
			self.conn = conn
		}
	}

	defer self.conn.Close()

	n, err := self.buffer.WriteTo(self.conn)
	if err != nil {
		Log("Write failed. size: %d, buf size: %d, error: %#v", n, self.buffer.Len(), err.Error())
		self.conn = nil
		return err
	}
	if n > 0 {
		Log("Forwarded: %d bytes (left: %d bytes)\n", n, self.buffer.Len())
	}

	self.conn = nil
	return nil

}

func (self *OutputForward) encodeRecordSet(ctx Context) error {
	v := []interface{}{ctx.tag, ctx.record}
	if self.enc == nil {
		self.enc = codec.NewEncoder(&self.buffer, self.codec)
	}
	err := self.enc.Encode(v)
	if err != nil {
		return err
	}
	return err
}

func init() {
	RegisterOutput("forward", func() interface{} {
		return new(OutputForward)
	})
}
