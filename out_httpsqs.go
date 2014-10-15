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

type outputHttpsqs struct {
	host string
	port int32

	auth           string
	flush_interval int
	debug          bool
	buffer         map[string][]byte
	client         *http.Client
}

func (self *outputHttpsqs) new() interface{} {

	return &outputHttpsqs{
		host:           "localhost",
		port:           1218,
		auth:           "testauth",
		flush_interval: 5,
		client:         &http.Client{},
		buffer:         make(map[string][]byte, 0),
	}
}

func (self *outputHttpsqs) configure(f map[string]interface{}) error {
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

func (self *outputHttpsqs) start(ctx chan Context) error {

	tick := time.NewTicker(time.Second * time.Duration(self.flush_interval))

	for {
		select {
		case <-tick.C:
			{
				if len(self.buffer) > 0 {
					self.flush()
				}
			}
		case s := <-ctx:
			{
				b, err := json.Marshal(s.record.data)
				if err != nil {
					continue
				}

				if len(self.buffer) == 0 {
					self.buffer[s.tag] = append(self.buffer[s.tag], byte('['))
				} else if len(self.buffer) > 0 {
					self.buffer[s.tag] = append(self.buffer[s.tag], byte(','))
				}

				self.buffer[s.tag] = append(self.buffer[s.tag], b...)
			}
		}
	}
}

func (self *outputHttpsqs) flush() {
	for k, v := range self.buffer {
		url := fmt.Sprintf("http://%s:%d/?name=%s&opt=put&auth=%s", self.host, self.port, k, self.auth)

		Log(url, string(v))

		v = append(v, byte(']'))

		resp, err := self.client.Post(url, "application/json", bytes.NewReader(v))
		if err != nil {
			Log(err)
			return
		}

		Log("resp:", *resp)

		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
		self.buffer[k] = self.buffer[k][0:0]
		delete(self.buffer, k)
	}
}

func init() {
	RegisterOutput("httpsqs", &outputHttpsqs{})
}
