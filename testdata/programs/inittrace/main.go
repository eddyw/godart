package main

import (
	"fmt"
	"runtime"
	"strings"
)

var first = inspect()
var second = inspect()

func inspect() int {
	var pcs [32]uintptr
	n := runtime.Callers(1, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	frame, more := frames.Next()
	if frame.Function != "main.inspect" || !more {
		panic("initializer callee frame")
	}
	frame, more = frames.Next()
	if frame.Function != "main.init" || !strings.HasSuffix(frame.File, "main.go") || !more {
		panic("initializer caller frame")
	}
	frame, _ = frames.Next()
	if !strings.HasPrefix(frame.Function, "runtime.") {
		panic("initializer runtime frame")
	}
	return 1
}

func main() {
	var pcs [32]uintptr
	n := runtime.Callers(1, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		if strings.HasPrefix(frame.Function, "runtime.doInit") {
			panic("initialization continues into main")
		}
		if !more {
			break
		}
	}
	fmt.Println(first + second)
}
