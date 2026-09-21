package main

import "fmt"

func external()

var events []int

func mark(n int) { events = append(events, n) }

func main() {
	if mark(1); false {
		external()
	} else {
		mark(2)
	}
	if mark(3); true {
		mark(4)
	} else {
		f := external
		f()
	}
	for mark(5); false; external() {
		external()
	}
	switch 2 {
	case 1:
		external()
	case 2:
		mark(6)
		fallthrough
	case 3:
		mark(7)
	default:
		external()
	}
	switch {
	case false:
		external()
	default:
		mark(8)
	}
	if fmt.Sprint(events) != "[1 2 3 4 5 6 7 8]" {
		panic(events)
	}
	fmt.Println("ok")
}
