package main

import (
	"fmt"
	"net"
	"time"
	"github.com/t-k/fluent-logger-golang/fluent"
)

type OutputForward struct {
	Host string
	Port int

	Send_timeout       int
	Heartbeat_interval int
}

func (self *OutputForward) New() interface{} {
	return &OutputForward{}
}
func (self *OutputForward) Configure(f map[string]interface{}) error {

	self.Host = f["host"].(string)
	self.Port = int(f["port"].(float64))
	self.Send_timeout = int(f["send_timeout"].(float64))
	self.Heartbeat_interval = int(f["heartbeat_interval"].(float64))

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
	tag := "debug.test"
	var logger *fluent.Fluent
	logger, err := fluent.New(fluent.Config{FluentPort: self.Port, FluentHost: self.Host})
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
