package main

type OutputStdout struct {
}

func (self *OutputStdout) new() interface{} {
	return &OutputStdout{}
}

func (self *OutputStdout) configure(f map[string]interface{}) error {
	return nil
}

func (self *OutputStdout) start(ctx chan Context) error {
	go func(ctx chan Context) {
		for {
			ch := <-ctx
			Log(ch.record)
		}
	}(ctx)

	return nil
}

func init() {
	RegisterOutput("stdout", &OutputStdout{})
}
