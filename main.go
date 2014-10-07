package main

type Context struct {
	tag  string
	data map[string]string
}

func main() {
	ctxInput := make(chan Context, 10)
	ctxOutput := make(chan Context, 10)

	NewInput(ctxInput)
	go NewOutput(ctxOutput)

	for {
		select {
		case ctx := <-ctxInput:
			{
				ctxOutput <- ctx
			}
		}
	}
}
