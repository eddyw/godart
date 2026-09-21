package base

import "example.com/initsuspend/state"

var Calls int
var A, B = pair()
var _ = ignored()
var Channel = ready()
var Received = <-Channel
var Callback = func() int { return wait(7) }
var Indirect = Callback()
var Boxed = wait(9)
var Address = &Boxed
var Literal = func() int { return wait(29) }()
var Recovered = recoverValue()

func wait(value int) int {
	ch := make(chan int)
	go func() { ch <- value }()
	return <-ch
}

func pair() (int, int) {
	Calls++
	defer state.Note("pair")
	return wait(11), wait(13)
}

func ignored() int {
	value := wait(17)
	state.Note("blank")
	return value
}

func ready() chan int {
	ch := make(chan int)
	go func() { ch <- 19 }()
	return ch
}

func init() {
	*Address += wait(1)
	state.Note("base init 1")
}

func init() {
	defer state.Note("base init 2")
	if wait(23) != 23 {
		panic("second wait")
	}
}

func recoverValue() (value int) {
	defer func() {
		if recover() != "initializer" {
			panic("initializer recovery")
		}
		value = wait(31)
	}()
	wait(1)
	panic("initializer")
}
