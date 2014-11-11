package main

type PluginHelper interface {
	Output(name string) (out Output, ok bool)
	Intput(name string) (in Input, ok bool)
}

type PipelineConfig struct {
	InputRunners  []interface{}
	OutputRunners []interface{}
	router        Router
}

func NewPipeLineConfig() *PipelineConfig {
	config := new(PipelineConfig)
	config.router.Init()
	return config
}

func (this *PipelineConfig) LoadConfig(path string) error {
	configure, _ := ParseConfig(nil, path)

	for _, v := range configure.Root.Elems {
		if v.Name == "source" {
			this.InputRunners = append(this.InputRunners, v.Attrs)
		} else if v.Name == "match" {
			v.Attrs["tag"] = v.Args
			this.OutputRunners = append(this.OutputRunners, v.Attrs)
		}
	}

	return nil
}
