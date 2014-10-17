package main

import (
	"bytes"
	"compress/gzip"
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
	gzip           bool
	buffer         map[string][]byte
	client         *http.Client
}

func (self *outputHttpsqs) new() interface{} {

	return &outputHttpsqs{
		host:           "localhost",
		port:           1218,
		auth:           "testauth",
		flush_interval: 5,
		gzip:           false,
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

	value = f["gzip"]
	if value != nil {
		self.gzip = value.(bool)
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

		v = append(v, byte(']'))
		Log("url:", url, ", buf length:", len(v))
		var buf bytes.Buffer
		var req *http.Request

		if self.gzip == true {
			gzw := gzip.NewWriter(&buf)
			gzw.Write([]byte(v))
			gzw.Close()
			req, _ = http.NewRequest("POST", url, bytes.NewReader(buf.Bytes()))
		} else {
			req, _ = http.NewRequest("POST", url, bytes.NewReader([]byte(v)))
		}

		req.Header.Add("Content-Encoding", "gzip")
		req.Header.Add("Content-Type", "application/json")

		resp, err := self.client.Do(req)
		if err != nil {
			Log("post failed:", err)
			continue
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
