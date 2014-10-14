package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

type OutputHttpsqs struct {
	host string
	port int32

	auth           string
	flush_interval int
	debug          bool
	buffer         []Context
	client         *http.Client
}

func (self *OutputHttpsqs) new() interface{} {

	return &OutputHttpsqs{
		host:           "localhost",
		port:           1218,
		auth:           "testauth",
		flush_interval: 5,
		client:         &http.Client{},
	}
}

func (self *OutputHttpsqs) configure(f map[string]interface{}) error {
	var value interface{}

	value = f["host"]
	if value != nil {
		self.host = value.(string)
	}

	value = f["port"]
	if value != nil {
		self.port = int32(value.(float64))
	}

	value = f["auth"]
	if value != nil {
		self.auth = value.(string)
	}

	value = f["flush_interval"]
	if value != nil {
		self.flush_interval = int(value.(float64))
	}

	return nil
}

func (self *OutputHttpsqs) start(ctx chan Context) error {

	tick := time.NewTicker(time.Second * time.Duration(self.flush_interval))

	for {
		select {
		case <-tick.C:
			{
				if len(self.buffer) > 0 {
					fmt.Println("flush ", len(self.buffer))
					self.flush()
				}
			}
		case s := <-ctx:
			{

				self.buffer = append(self.buffer, s)
			}
		}
	}
}

func (self *OutputHttpsqs) flush() {
	for _, v := range self.buffer {
		url := fmt.Sprintf("http://%s:%d/?name=%s&opt=put&auth=%s", self.host, self.port, v.tag, self.auth)
		b, err := json.Marshal(v.record.data)
		if err != nil {
			continue
		}

		resp, err := self.client.Post(url, "application/json", bytes.NewBuffer(b))
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(resp)
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}

	self.buffer = self.buffer[0:0]

}

func init() {
	RegisterOutput("httpsqs", &OutputHttpsqs{})
}
