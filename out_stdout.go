package main

type OutputStdout struct {
}

func (self *OutputStdout) Init(f map[string]string) error {
	return nil
}

func (self *OutputStdout) Run(runner OutputRunner) error {

	pack := <-runner.InChan()
	Log(pack.Msg)
	pack.Recycle()

	return nil
}

func init() {
	RegisterOutput("stdout", func() interface{} {
		return new(OutputStdout)
	})
}
