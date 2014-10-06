package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
  Inputs_config []interface{}
  Outputs_config []interface{}
}

var config Config

func ReadConf(filePath string) (interface{}) {
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}

	decoder := json.NewDecoder(file)

  var jsontype interface{}
	err = decoder.Decode(&jsontype)
	if err != nil {
		fmt.Println("Decode error: ", err)
    panic(err)
  }

	return jsontype
}

func init() {
	args := ReadConf("config.json").(map[string]interface{})
  fmt.Println(args)

  config.Inputs_config = args["sources"].([]interface{})
  config.Outputs_config = args["matches"].([]interface{})

}
