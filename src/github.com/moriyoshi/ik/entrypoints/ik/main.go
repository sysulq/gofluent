package main

import (
	"flag"
	"fmt"
	"github.com/moriyoshi/ik"
	"github.com/moriyoshi/ik/plugins"
	"github.com/moriyoshi/ik/parsers"
	"log"
	"os"
	"path"
	"errors"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(255)
}

func configureScoreboards(logger *log.Logger, registry *MultiFactoryRegistry, engine ik.Engine, config *ik.Config) error {
       for _, v := range config.Root.Elems {
	       switch v.Name {
		case "scoreboard":
			type_ := v.Attrs["type"]
			scoreboardFactory := registry.LookupScoreboardFactory(type_)
			if scoreboardFactory == nil {
				return errors.New("Could not find scoreboard factory: " + type_)
			}
			scoreboard, err := scoreboardFactory.New(engine, registry, v)
			if err != nil {
				return err
			}
			err = engine.Launch(scoreboard)
			if err != nil {
				return err
			}
			logger.Printf("Scoreboard plugin loaded: %s", scoreboardFactory.Name())
		}
	}
	return nil
}

func main() {
	logger := log.New(os.Stdout, "[ik] ", log.Lmicroseconds)

	var config_file string
	var help bool
	flag.StringVar(&config_file, "c", "/etc/fluent/fluent.conf", "config file path (default: /etc/fluent/fluent.conf)")
	flag.BoolVar(&help, "h", false, "show help")
	flag.Parse()

	if help || config_file == "" {
		usage()
	}

	dir, file := path.Split(config_file)
	opener := ik.DefaultOpener(dir)
	config, err := ik.ParseConfig(opener, file)
	if err != nil {
		println(err.Error())
		return
	}

	scorekeeper := ik.NewScorekeeper(logger)

	registry := NewMultiFactoryRegistry(scorekeeper)

	for _, _plugin := range plugins.GetPlugins() {
		switch plugin := _plugin.(type) {
		case ik.InputFactory:
			registry.RegisterInputFactory(plugin)
		case ik.OutputFactory:
			registry.RegisterOutputFactory(plugin)
		}
	}

	for _, _plugin := range parsers.GetPlugins() {
		registry.RegisterLineParserPlugin(_plugin)
	}

	registry.RegisterScoreboardFactory(&HTMLHTTPScoreboardFactory {})

	router := ik.NewFluentRouter()
	engine := ik.NewEngine(logger, opener, registry, scorekeeper, router)
	defer func () {
		err := engine.Dispose()
		if err != nil {
			engine.Logger().Println(err.Error())
		}
	}()

	err = ik.NewFluentConfigurer(logger, registry, registry, router).Configure(engine, config)
	if err != nil {
		println(err.Error())
		return
	}
	err = configureScoreboards(logger, registry, engine, config)
	if err != nil {
		println(err.Error())
		return
	}
	engine.Start()
}

// vim: sts=4 sw=4 ts=4 noet
