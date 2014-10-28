package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
)

type Config struct {
	Inputs_config  []interface{}
	Outputs_config []interface{}
}

var config Config

func ReadConf(filePath *string) interface{} {
	file, err := os.Open(*filePath)
	if err != nil {
		log.Fatal(err)
	}

	decoder := json.NewDecoder(file)

	var jsontype interface{}
	err = decoder.Decode(&jsontype)
	if err != nil {
		log.Fatal("Decode error: ", err)
	}

	return jsontype
}

func init() {
	c := flag.String("config", "config.json", "config filepath")
	flag.Parse()

	args := ReadConf(c).(map[string]interface{})

	config.Inputs_config = args["sources"].([]interface{})
	config.Outputs_config = args["matches"].([]interface{})

}
