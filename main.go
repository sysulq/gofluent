package main

import (
	"flag"
	"log"
	"os"
	"runtime/pprof"
)

type Context struct {
	tag    string
	record Record
}

type Record struct {
	timestamp int64
	data      map[string]string
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

	config := NewPipeLineConfig()
	config.LoadConfig(*c)

	Run(config)
}
