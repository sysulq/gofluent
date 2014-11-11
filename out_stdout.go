package main

type OutputStdout struct {
}

func (self *OutputStdout) Init(f map[string]string) error {
	return nil
}

func (self *OutputStdout) Run(ctx chan Context) error {
	go func(ctx chan Context) {
		for {
			ch := <-ctx
			Log(ch.record)
		}
	}(ctx)

	return nil
}

func init() {
	RegisterOutput("stdout", func() interface{} {
		return new(OutputStdout)
	})
}
