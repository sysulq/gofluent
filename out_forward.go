package main

import (
	"bytes"
	"github.com/ugorji/go/codec"
	"log"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"time"
)

type OutputForward struct {
	host string
	port int

	connect_timeout    int
	flush_interval     int
	sync_interval      int
	buffer_queue_limit int64
	buffer_chunk_limit int64

	buffer_path string

	codec      *codec.MsgpackHandle
	enc        *codec.Encoder
	conn       net.Conn
	msg_buffer bytes.Buffer
	buffer     bytes.Buffer
	backend    BackendQueue
}

func (self *OutputForward) Init(config map[string]string) error {
	_codec := codec.MsgpackHandle{}
	_codec.MapType = reflect.TypeOf(map[string]interface{}(nil))
	_codec.RawToString = false
	_codec.StructToArray = true

	self.host = "localhost"
	self.port = 8888
	self.flush_interval = 10
	self.sync_interval = 2
	self.buffer_path = "/tmp/test"
	self.buffer_queue_limit = 64 * 1024 * 1024
	self.buffer_chunk_limit = 8 * 1024 * 1024
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

	value = config["sync_interval"]
	if len(value) > 0 {
		sync_interval, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		self.sync_interval = sync_interval
	}

	value = config["buffer_path"]
	if len(value) > 0 {
		self.buffer_path = value
	}

	value = config["buffer_queue_limit"]
	if len(value) > 0 {
		buffer_queue_limit, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		self.buffer_queue_limit = int64(buffer_queue_limit) * 1024 * 1024
	}

	value = config["buffer_chunk_limit"]
	if len(value) > 0 {
		buffer_chunk_limit, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		self.buffer_chunk_limit = int64(buffer_chunk_limit) * 1024 * 1024
	}
	return nil
}

func (self *OutputForward) Run(runner OutputRunner) error {
	l := log.New(os.Stderr, "", log.LstdFlags)

	sync_interval := time.Duration(self.sync_interval)
	base := filepath.Base(self.buffer_path)
	dir := filepath.Dir(self.buffer_path)
	self.backend = newDiskQueue(base, dir, self.buffer_queue_limit, 2500, sync_interval*time.Second, l)

	tick := time.NewTicker(time.Second * time.Duration(self.flush_interval))

	for {
		select {
		case <-tick.C:
			{
				if self.backend.Depth() > 0 {
					log.Printf("flush %d left", self.backend.Depth())
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
			log.Println("net.DialTimeout failed, err", err)
			return err
		} else {
			self.conn = conn
		}
	}

	defer self.conn.Close()

	count := 0
	depth := self.backend.Depth()

	if self.buffer.Len() == 0 {
		for i := int64(0); i < depth; i++ {
			self.buffer.Write(<-self.backend.ReadChan())
			count++
			if int64(self.buffer.Len()) > self.buffer_chunk_limit {
				break
			}
		}
	}

	log.Println("buffer sent:", self.buffer.Len(), "count:", count)
	n, err := self.buffer.WriteTo(self.conn)
	if err != nil {
		log.Printf("Write failed. size: %d, buf size: %d, error: %#v", n, self.buffer.Len(), err.Error())
		self.conn = nil
		return err
	}
	if n > 0 {
		log.Printf("Forwarded: %d bytes (left: %d bytes)\n", n, self.buffer.Len())
	}

	self.conn = nil

	return nil

}

func (self *OutputForward) encodeRecordSet(msg Message) error {
	v := []interface{}{msg.Tag, msg.Timestamp, msg.Data}
	if self.enc == nil {
		self.enc = codec.NewEncoder(&self.msg_buffer, self.codec)
	}
	err := self.enc.Encode(v)
	if err != nil {
		return err
	}
	self.backend.Put(self.msg_buffer.Bytes())
	self.msg_buffer.Reset()
	return err
}

func init() {
	RegisterOutput("forward", func() interface{} {
		return new(OutputForward)
	})
}
