package main

import (
	"fmt"
 "regexp"
)

type FluentdCtx struct {
  tag    string
	data   map[string]string
}

func main() {
  ctxInput := make(chan FluentdCtx, 10)
	ctxOutput := make(chan FluentdCtx, 10)

	NewInput(ctxInput)
  NewOutput(ctxOutput)

	for {
    select {
      case ctx := <-ctxInput:
        for _, output_config := range config.Outputs_config {
          fmt.Println("ctxInput")
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
