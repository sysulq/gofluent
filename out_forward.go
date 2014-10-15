package main

import (
	"bytes"
	"fmt"
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

	logger *log.Logger
	codec  *codec.MsgpackHandle
	enc    *codec.Encoder
	conn   net.Conn
	buffer bytes.Buffer
}

func (self *OutputForward) new() interface{} {
	_codec := codec.MsgpackHandle{}
	_codec.MapType = reflect.TypeOf(map[string]interface{}(nil))
	_codec.RawToString = false
	_codec.StructToArray = true

	return &OutputForward{
		host:            "localhost",
		port:            8888,
		flush_interval:  5,
		connect_timeout: 10,
		codec:           &_codec,
		logger:          log.New(os.Stderr, "[journal] ", 0),
	}
}

func (self *OutputForward) configure(f map[string]interface{}) error {
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

func (self *OutputForward) start(ctx chan Context) error {

	tick := time.NewTicker(time.Second * time.Duration(self.flush_interval))

	for {
		select {
		case <-tick.C:
			{
				if self.buffer.Len() > 0 {
					fmt.Println("flush ", self.buffer.Len())
					self.flush()
				}
			}
		case s := <-ctx:
			{
				self.encodeRecordSet(s)
			}
		}
	}

}

func (self *OutputForward) flush() error {
	if self.conn == nil {
		conn, err := net.DialTimeout("tcp", self.host+":"+strconv.Itoa(self.port), time.Second*time.Duration(self.connect_timeout))
		if err != nil {
			self.logger.Printf("%#v", err.Error())
			return err
		} else {
			self.conn = conn
		}
	}

	defer self.conn.Close()

	n, err := self.buffer.WriteTo(self.conn)
	if err != nil {
		self.logger.Printf("Write failed. size: %d, buf size: %d, error: %#v", n, self.buffer.Len(), err.Error())
		self.conn = nil
		return err
	}
	if n > 0 {
		self.logger.Printf("Forwarded: %d bytes (left: %d bytes)\n", n, self.buffer.Len())
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
	RegisterOutput("forward", &OutputForward{})
}
