package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

type outputHttpsqs struct {
	host string
	port int

	auth           string
	flush_interval int
	debug          bool
	gzip           bool
	buffer         map[string][]byte
	client         *http.Client
	count          int
}

func (self *outputHttpsqs) new() interface{} {

	return &outputHttpsqs{
		host:           "localhost",
		port:           1218,
		flush_interval: 10,
		gzip:           true,
		client:         &http.Client{},
		buffer:         make(map[string][]byte, 0),
	}
}

func (self *outputHttpsqs) configure(f map[string]string) error {
	var value string

	value = f["host"]
	if len(value) > 0 {
		self.host = value
	}

	value = f["port"]
	if len(value) > 0 {
		self.port, _ = strconv.Atoi(value)
	}

	value = f["auth"]
	if len(value) > 0 {
		self.auth = value
	}

	value = f["flush_interval"]
	if len(value) > 0 {
		self.flush_interval, _ = strconv.Atoi(value)
	}

	value = f["gzip"]
	if len(value) > 0 {
		if value == "off" {
			self.gzip = false
		}
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

				self.count++
				self.buffer[s.tag] = append(self.buffer[s.tag], b...)
			}
		}
	}
}

func (self *outputHttpsqs) flush() {
	for k, v := range self.buffer {
		url := fmt.Sprintf("http://%s:%d/?name=%s&opt=put&auth=%s", self.host, self.port, k, self.auth)

		v = append(v, byte(']'))
		Log("url:", url, "count:", self.count, "buf length:", len(v))
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

		Log("StatusCode:", resp.StatusCode, "Pos:", resp.Header.Get("Pos"))

		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
		self.buffer[k] = self.buffer[k][0:0]
		delete(self.buffer, k)
		self.count = 0
	}
}

func init() {
	RegisterOutput("httpsqs", &outputHttpsqs{})
}
