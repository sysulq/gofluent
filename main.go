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

type Configure struct {
	Inputs_config  []interface{}
	Outputs_config []interface{}
}

var config Configure
var router Router

func init() {
	c := flag.String("c", "gofluent.conf", "config filepath")

	configure, _ := ParseConfig(nil, *c)

	for _, v := range configure.Root.Elems {
		if v.Name == "source" {
			config.Inputs_config = append(config.Inputs_config, v.Attrs)
		} else if v.Name == "match" {
			v.Attrs["tag"] = v.Args
			config.Outputs_config = append(config.Outputs_config, v.Attrs)
		}
	}

	router.Init()
}

func main() {
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

	ctxInput := make(chan Context, 10)
	ctxOutput := make(chan Context, 10)

	go NewInputs(ctxInput)
	go NewOutputs(ctxOutput)

	select {}
}
