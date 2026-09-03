package main

import (
	"fmt"
	"runtime"
	"strings"
)

// stackFacts reads the trace of the goroutine that runs now. The 2 runtimes
// write a different text, thus the program prints only the facts that hold on
// each one.
func stackFacts() {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	text := string(buf[:n])
	fmt.Println("stack bytes", n > 0)
	fmt.Println("stack head", strings.HasPrefix(text, "goroutine "))
	fmt.Println("stack holds main", strings.Contains(text, "main.stackFacts"))
	fmt.Println("stack lines", strings.Count(text, "\n") >= 3)
}

// allFacts reads the trace of each goroutine while 1 goroutine waits.
func allFacts(wait chan int) {
	buf := make([]byte, 8192)
	n := runtime.Stack(buf, true)
	text := string(buf[:n])
	fmt.Println("all heads", strings.Count(text, "goroutine ") >= 2)
	fmt.Println("all live", runtime.NumGoroutine() >= 2)
}

// exitFacts proves that Goexit runs each deferred call and ends the goroutine.
func exitFacts() {
	done := make(chan string, 2)
	go func() {
		defer func() { done <- "deferred" }()
		runtime.Goexit()
		done <- "not reached"
	}()
	fmt.Println("goexit", <-done)
}

// nilPanic proves that a panic of a nil value gives the error of Go.
func nilPanic() {
	defer func() {
		value := recover()
		err, ok := value.(*runtime.PanicNilError)
		fmt.Println("panic nil", ok, err.Error())
	}()
	panic(nil)
}

func main() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	fmt.Println("cgo calls", runtime.NumCgoCall())
	fmt.Println("compiler", len(runtime.Compiler) > 0)
	fmt.Println("version", len(runtime.Version()) > 0)

	value := []int{1, 2, 3}
	runtime.KeepAlive(value)
	fmt.Println("kept", len(value))

	stackFacts()

	wait := make(chan int)
	go func() { <-wait }()
	runtime.Gosched()
	allFacts(wait)
	close(wait)

	exitFacts()
	nilPanic()
	fmt.Println("done")
}
