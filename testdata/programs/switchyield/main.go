package main

import (
	"fmt"
	"runtime"
)

var calls []int

func value(n int) int {
	runtime.Gosched()
	calls = append(calls, n)
	return n
}

func main() {
	switch value(2) {
	case value(1), value(2), value(3):
		fmt.Println("match", calls)
	default:
		panic("no match")
	}
	calls = nil
	switch {
	case value(1) == 0:
		panic("first case")
	case value(2) == 2 || value(3) == 3:
		fmt.Println("condition", calls)
	default:
		panic("no condition")
	}
	calls = nil
	switch value(4) {
	case value(1):
		panic("wrong value")
	default:
		fmt.Println("default", calls)
	}
}
