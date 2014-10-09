package main

import (
	"fmt"
	"github.com/t-k/fluent-logger-golang/fluent"
	"net"
	"time"
)

type OutputForward struct {
	host string
	port int

	send_timeout       int
	heartbeat_interval int
}

func (self *OutputForward) new() interface{} {
	return &OutputForward{ 
		host: "localhost",
		port: 8888,
		send_timeout: 3,
		heartbeat_interval: 1}
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

	value = f["heartbeat_interval"]
	if value != nil {
		self.heartbeat_interval = int(value.(float64))
	}

	value = f["send_timeout"]
	if value != nil {
		self.send_timeout = int(value.(float64))
	}

	return nil
}

func (self *OutputForward) start(ctx chan Context) error {
	down := make(chan bool, 1)
	go self.toFluent(ctx, down)

	tick := time.NewTicker(time.Second * time.Duration(self.heartbeat_interval))

	for {
		<-tick.C
		fmt.Println("doHealthcheck")
		self.doHeartbeat(down)
	}

}

func (self *OutputForward) toFluent(ctx chan Context, down chan bool) {
	var logger *fluent.Fluent
	logger, err := fluent.New(fluent.Config{FluentPort: self.port, FluentHost: self.host})
	if err != nil {
		panic(err)
	}

	for {
		select {
		case s := <-ctx:
			{
				logger.Post(s.tag, s.data)
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

func (self *OutputForward) doHeartbeat(down chan bool) {
	udpAddr := fmt.Sprintf("%s:%d", self.host, self.port)
	serverAddr, err := net.ResolveUDPAddr("udp", udpAddr)
	if err != nil {
		panic(err)
	}

	c, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		panic(err)
	}

	defer c.Close()

	c.SetDeadline(time.Now().Add(time.Second * time.Duration(self.send_timeout)))
	c.Write([]byte("t"))

	b := make([]byte, 1)
	if _, err := c.Read(b); err != nil {
		down <- true
		return
	}

	down <- false
}

func init() {
	RegisterOutput("forward", &OutputForward{})
}
