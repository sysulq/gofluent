package main

type Context struct {
	tag  string
	record Record
}

type Record struct {
  timestamp int64
  data      map[string]string
}

func main() {
	ctxInput := make(chan Context, 10)
	ctxOutput := make(chan Context, 10)

	go NewInputs(ctxInput)
	go NewOutputs(ctxOutput)

	for {
		select {
		case ctx := <-ctxInput:
			{
				ctxOutput <- ctx
			}
		}
	}
}
