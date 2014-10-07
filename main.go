package main

import (
	//"fmt"
 "regexp"
)

type Context struct {
  tag    string
	data   map[string]string
}

func main() {
  ctxInput := make(chan Context, 10)
	ctxOutput := make(chan Context, 10)

	NewInput(ctxInput)
  NewOutput(ctxOutput)

	for {
    select {
      case ctx := <-ctxInput:
        for _, output_config := range config.Outputs_config {

          f := output_config.(map[string]interface{})
          tag := f["tag"].(string)

          tagre := regexp.MustCompile(tag)

          flag := tagre.MatchString(ctx.tag)
          if flag == false {
            continue
          }

          ctxOutput <- ctx
        }
    }
	}
}
