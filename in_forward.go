package main

import (
	"log"
	"net"
	"sync"
)

type InputForward struct {
	Host string
	Port string
}

func (this *InputForward) Init(cf map[string]string) error {
	value := cf["bind"]
	if len(value) > 0 {
		this.Host = value
	} else {
		log.Panicln("No bind info configured.")
	}

	value = cf["port"]
	if len(value) > 0 {
		this.Port = value
	} else {
		log.Panicln("No port info configured.")
	}

	return nil
}

func (this *InputForward) Run(runner iRunner) error {
	var listener net.Listener
	var conn net.Conn
	var err error
	var wg sync.WaitGroup

	for {
		if conn, err = listener.Accept(); err != nil {
			if err.(net.Error).Temporary() {
				log.Println("TCP accept failed:", err)
				continue
			} else {
				break
			}

			wg.Add(1)

			go this.handleConn(conn, wg)
		}
	}

	wg.Wait()
	return nil
}

func (this *InputForward) handleConn(conn net.Conn, wg sync.WaitGroup) {
	defer wg.Done()

}

func init() {
	RegisterInput("forward", func() interface{} {
		return new(InputForward)
	})
}
