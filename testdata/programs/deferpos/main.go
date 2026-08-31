package main

import (
	"fmt"
	"runtime"
)

// caller gives the line of the function that runs its deferred calls now.
func caller() int {
	_, _, line, _ := runtime.Caller(2)
	return line
}

var lines []int

func report() { lines = append(lines, caller()) }

func explicitReturn() int {
	defer report()
	return 42
}

func fallOff() {
	defer report()
	fmt.Sprint("x")
}

func explicitReturnM(ch chan int) int {
	defer report()
	ch <- 1
	return 43
}

func fallOffM(ch chan int) {
	defer report()
	ch <- 2
}

func deferBlocksM(ch chan int) {
	defer func() {
		ch <- 3
		lines = append(lines, caller())
	}()
	fmt.Sprint("y")
}

func main() {
	ch := make(chan int, 8)
	fmt.Println(explicitReturn())
	fallOff()
	fmt.Println(explicitReturnM(ch))
	fallOffM(ch)
	deferBlocksM(ch)
	fmt.Println(lines)
}
