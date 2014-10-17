package main

import (
	"flag"
	"runtime"
	"time"
)

type Context struct {
	tag    string
	record Record
}

type Record struct {
	timestamp int64
	data      map[string]string
}

func GoRuntimeStats() {
	m := &runtime.MemStats{}
	runtime.ReadMemStats(m)

	Log("# goroutines: ", runtime.NumGoroutine(), "Memory Acquired: ", m.Sys, "Memory Used: ", m.Alloc)
}

func main() {
	interval := flag.Int("stats", 600, "stats interval")

	flag.Parse()

	ctxInput := make(chan Context, 10)
	ctxOutput := make(chan Context, 10)

	go NewInputs(ctxInput)
	go NewOutputs(ctxOutput)

	tick := time.NewTicker(time.Second * time.Duration(*interval))

	for {
		select {
		case <-tick.C:
			{
				GoRuntimeStats()
			}
		case ctx := <-ctxInput:
			{
				ctxOutput <- ctx
			}
		}
	}
}
