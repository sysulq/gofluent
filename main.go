package main

import (
	"flag"
	"log"
	"os"
	"runtime/pprof"
)

type GlobalConfig struct {
	PoolSize int
}

func main() {
	c := flag.String("c", "gofluent.conf", "config filepath")
	p := flag.String("p", "", "write cpu profile to file")
	flag.Parse()

	if *p != "" {
		f, err := os.Create(*p)
		if err != nil {
			log.Fatalln(err)
		}
		pprof.StartCPUProfile(f)
		defer func() {
			pprof.StopCPUProfile()
			f.Close()
		}()
	}

	gc := DefaultGC()
	config := NewPipeLineConfig(gc)
	config.LoadConfig(*c)

	Run(config)
}
