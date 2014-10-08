package main

import (
	"fmt"
	"github.com/t-k/fluent-logger-golang/fluent"
	"net"
	"time"
)

type OutputForward struct {
	Host string
	Port int

	Send_timeout       int
	Heartbeat_interval int
}

func (self *OutputForward) New() interface{} {
	return &OutputForward{ 
		Host: "localhost",
		Port: 8888,
		Send_timeout: 3,
		Heartbeat_interval: 1}
}
func (self *OutputForward) Configure(f map[string]interface{}) error {
	var value interface{}

	value = f["host"]
	if value != nil {
		self.Host = value.(string)
	}

	value = f["port"]
	if value != nil {
		self.Port = int(value.(float64))
	}

	value = f["heartbeat_interval"]
	if value != nil {
		self.Heartbeat_interval = int(value.(float64))
	}

	value = f["send_timeout"]
	if value != nil {
		self.Send_timeout = int(value.(float64))
	}

	return nil
}

func (self *OutputForward) Start(ctx chan Context) error {
	down := make(chan bool, 1)
	go self.toFluent(ctx, down)

	tick := time.NewTicker(time.Second * time.Duration(self.Heartbeat_interval))

	for {
		<-tick.C
		fmt.Println("doHealthcheck")
		self.doHeartbeat(down)
	}

}

func (self *OutputForward) toFluent(ctx chan Context, down chan bool) {
	var logger *fluent.Fluent
	logger, err := fluent.New(fluent.Config{FluentPort: self.Port, FluentHost: self.Host})
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
	udpAddr := fmt.Sprintf("%s:%d", self.Host, self.Port)
	serverAddr, err := net.ResolveUDPAddr("udp", udpAddr)
	if err != nil {
		panic(err)
	}

	c, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		panic(err)
	}

	defer c.Close()

	c.SetDeadline(time.Now().Add(time.Second * time.Duration(self.Send_timeout)))
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
