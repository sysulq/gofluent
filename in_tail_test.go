package main

import (
        "testing"
	"fmt"
	"encoding/json"
)

func Test_new(t *testing.T) {
  in := new(inputTail)
  intail := in.new().(*inputTail)

  intail.format = "json"
  
  var st map[string]string
  data := []byte(`{"test":"mytest","data":"mydata"}`)

  err := json.Unmarshal(data, &st)
  if err != nil {
    t.Log("fail to Unmarshal")
  }

  fmt.Println(st)
}

