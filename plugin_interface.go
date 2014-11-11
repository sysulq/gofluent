package main

type Output interface {
	Init(map[string]string) error
	Run(ctx chan Context) error
}

type Input interface {
	Init(map[string]string) error
	Run(ctx chan Context) error
}
