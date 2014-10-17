package main

import (
	"bufio"
	"flag"
	"log"
	"os"
)

var (
	logFileName string
	logChannel  chan interface{}
)

func init() {
	logFileName := flag.String("log", "error.log", "log filepath")

	logChannel = make(chan interface{})

	go func(logChannel <-chan interface{}) {
		for msg := range logChannel {
			f, err := os.OpenFile(*logFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, os.ModePerm)
			if err != nil {
				// TODO: If this panics it would break the goroutine, would nothing
				// be logged ever again?
				panic(err)
			}

			w := bufio.NewWriter(f)
			log.SetOutput(w)
			log.Println(msg)

			w.Flush()
			f.Close()

			log.SetOutput(os.Stdout)
			log.Println(msg)
		}
	}(logChannel)
}

func Log(message ...interface{}) {
	logChannel <- message
}
