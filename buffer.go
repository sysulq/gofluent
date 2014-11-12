package main

type Buffer struct {
	data           []byte
	count          int
	flush_interval int
}

func (self *Buffer) Push() {

}
