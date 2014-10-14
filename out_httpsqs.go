package main

import (
	"fmt"
	//"time"
	"encoding/json"
	httpsqs "github.com/crosstime1986/go-httpsqs"
)

type OutputHttpsqs struct {
	host string
	port int32

	auth           string
	flush_interval int
	debug          bool
}

func (self *OutputHttpsqs) new() interface{} {

	return &OutputHttpsqs{
		host:           "localhost",
		port:           1218,
		auth:           "testauth",
		flush_interval: 5,
		debug:          false,
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

	value = f["auth"]
	if value != nil {
		self.flush_interval = int(value.(float64))
	}

	value = f["debug"]
	if value != nil {
		self.debug = value.(bool)
	}

	return nil
}

func (self *OutputHttpsqs) start(ctx chan Context) error {

	q := httpsqs.NewClient(self.host, self.port, self.auth, self.debug)

	for {
		select {
		case s := <-ctx:
			{
				b, err := json.Marshal(s.record.data)
				if err != nil {
					continue
				}

				res, err := q.Puts(s.tag, string(b))
				if err != nil || res != "HTTPSQS_PUT_OK" {
					fmt.Println(err)
				}
			}
		}
	}

}

func init() {
	RegisterOutput("httpsqs", &OutputHttpsqs{})
}
