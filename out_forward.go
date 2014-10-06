package main

import (
	"os"
	"fmt"
	"net"
	"time"
	"encoding/json"
	"github.com/t-k/fluent-logger-golang/fluent"
)

type configure struct {
	Host string `json:"host"`
	Port int    `json:"port"`

	Send_timeout       int `json:"send_timeout"`
	Heartbeat_interval int `json:"heartbeat_interval"`
}

type OutputForward struct {
	config configure
}

func (self *OutputForward) Configure(i int, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}

	decoder := json.NewDecoder(file)
	tmp := make([]configure, 1)

	err = decoder.Decode(&tmp)
	if err != nil {
		fmt.Println("Decode error: ", err)
	}

	self.config = tmp[i]

	return err
}

func (self *OutputForward) Start(ctx chan FluentdCtx) error {
	down := make(chan bool, 1)
	go self.toFluent(ctx, down)

	tick := time.NewTicker(time.Second * time.Duration(self.config.Heartbeat_interval))

	for {
      	<-tick.C
    	fmt.Println("doHealthcheck")
    	self.doHeartbeat(&config, down)
	}

}

func (self *OutputForward) toFluent(ctx chan FluentdCtx, down chan bool) {
	tag := "debug.test"
	var logger *fluent.Fluent
	logger, err := fluent.New(fluent.Config{FluentPort: self.config.Port, FluentHost: self.config.Host})
	if err != nil {
		panic(err)
	}

	for {
		select {
		case s := <-ctx:
			{
				logger.Post(tag, s.data)
			}
		case s := <-down:
			{
				if s == true {
					fmt.Println("down")
					logger.Close()
				} else {
					fmt.Println("up")
				}
			}
		}
	}
}

func (self *OutputForward) doHeartbeat(config *Config, down chan bool) {
	udpAddr := fmt.Sprintf("%s:%d", self.config.Host, self.config.Port)
	serverAddr, err := net.ResolveUDPAddr("udp", udpAddr)
	if err != nil {
		panic(err)
	}

	c, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		panic(err)
	}

	defer c.Close()

	c.SetDeadline(time.Now().Add(time.Second * time.Duration(self.config.Send_timeout)))
	c.Write([]byte("t"))

	b := make([]byte, 1)
	if _, err := c.Read(b); err != nil {
		down <- true
		return
	}

	down <- false
}

func NewOutputForward() *OutputForward {
	return &OutputForward{}
}

func init() {
	RegisterOutput("forward", NewOutputForward())
}
