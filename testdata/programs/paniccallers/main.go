package main

import (
	"fmt"
	"runtime"
	"strings"
)

func trace(label string) {
	pcs := make([]uintptr, 20)
	pcs = pcs[:runtime.Callers(2, pcs)]
	frames := runtime.CallersFrames(pcs)
	var names []string
	for {
		frame, more := frames.Next()
		names = append(names, strings.TrimPrefix(frame.Function, "main."))
		if !more || frame.Function == "main.main" {
			break
		}
	}
	fmt.Println(label, strings.Join(names, ","))
}

func suspended(ch chan int) {
	defer func() { trace("after") }()
	defer func() {
		if recover() != "first" {
			panic("recovery")
		}
		<-ch
		trace("recovered")
	}()
	panic("first")
}

func nested() {
	defer func() {
		defer func() {
			trace("nested")
			if recover() != "second" {
				panic("nested recovery")
			}
		}()
		if recover() != "first" {
			panic("outer recovery")
		}
		panic("second")
	}()
	panic("first")
}

func delayed(ch chan int) {
	<-ch
	defer func() {
		trace("delayed")
		if recover() != "late" {
			panic("late recovery")
		}
	}()
	panic("late")
}

func sequence(yield func(int) bool) { yield(1) }

func ranges() {
	before := func() { trace("before range") }
	before()
	for range sequence {
		trace("range")
		inner := func() { trace("inside range") }
		inner()
		for range sequence {
			trace("nested range")
		}
	}
	after := func() { trace("after range") }
	after()
}

func main() {
	ch := make(chan int)
	go func() { ch <- 1 }()
	suspended(ch)
	nested()
	go func() { ch <- 2 }()
	delayed(ch)
	ranges()
	trace("normal")
}
