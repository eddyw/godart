package main

import (
	"fmt"

	"example.com/initsuspend/base"
	"example.com/initsuspend/state"
)

var value = initialize()

func initialize() int {
	state.Note("main variable")
	ch := make(chan int)
	go func() { ch <- base.A + base.B }()
	return <-ch
}

func init() {
	ch := make(chan bool)
	go func() { ch <- true }()
	<-ch
	value++
	state.Note("main init")
}

func main() {
	fmt.Println(value, base.Calls, base.Received, base.Indirect, base.Boxed, base.Literal, base.Recovered)
	fmt.Println(state.Steps)
}
