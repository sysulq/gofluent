package main

import (
	"fmt"
 "regexp"
)

type FluentdCtx struct {
	source Source
	match  Match
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
        for i := range config.Matches {
          fmt.Println("ctxInput")
          tagre := regexp.MustCompile(config.Matches[i].Tag)

          flag := tagre.MatchString(ctx.source.Tag)
          if flag == false {
            continue
          }

          ctxOutput <- ctx
        }
    }
	}
}
