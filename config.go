package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type Source struct {
	Type   string `json:"type"`
	Path   string `json:"path"`
	Format string `json:"format"`
	Tag    string `json:"tag"`
}

type Match struct {
	Tag  string `json:"tag"`
	Type string `json:"type"`

}

type Config struct {
	Sources []Source `json:"sources"`
	Matches []Match  `json:"matches"`
}

var config Config

func ReadConf(filePath string) *Config {
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}

	decoder := json.NewDecoder(file)

	err = decoder.Decode(&config)
	if err != nil {
		fmt.Println("Decode error: ", err)
		os.Exit(1)
	}

	return &config
}

func init() {
	config = *ReadConf("config.json")
  fmt.Println(config)
}
