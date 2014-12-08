package main

import (
	"flag"
	"io"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
)

type GlobalConfig struct {
	PoolSize int
}

func DefaultGC() *GlobalConfig {
	gc := new(GlobalConfig)
	gc.PoolSize = 1000
	return gc
}

func main() {
	c := flag.String("c", "gofluent.conf", "config filepath")
	p := flag.String("p", "", "write cpu profile to file")
	v := flag.String("v", "error.log", "log file path")
	flag.Parse()

	f, err := os.OpenFile(*v, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("os.Open failed, err:", err)
	}
	defer f.Close()

	multi := io.MultiWriter(f, os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.SetOutput(multi)

	if *p != "" {
		go func() {
			log.Println(http.ListenAndServe("0.0.0.0:"+*p, nil))
		}()
	}

	gc := DefaultGC()
	config := NewPipeLineConfig(gc)
	config.LoadConfig(*c)

	Run(config)
}
