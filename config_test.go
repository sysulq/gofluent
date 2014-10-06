package main

import (
	"testing"
)

func Test_ReadConf(t *testing.T) {
	config := ReadConf("config.json")
	t.Log("ReadConf passed: ", config)
}
