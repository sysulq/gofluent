package main

type Input interface {
	Init(config map[string]string) error
	Run(in InputRunner) error
}

type Output interface {
	Init(config map[string]string) error
	Run(out OutputRunner) error
}
