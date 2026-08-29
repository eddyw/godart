package main

import (
	"fmt"
	"path/filepath"
	"runtime"
)

func inner() {
	pc, file, line, ok := runtime.Caller(0)
	fmt.Println("caller 0", filepath.Base(file), line, ok)
	fmt.Println("name", runtime.FuncForPC(pc).Name())
	_, file, line, _ = runtime.Caller(1)
	fmt.Println("caller 1", filepath.Base(file), line)
}

func outer() {
	inner()
}

func walk() {
	pc := make([]uintptr, 8)
	n := runtime.Callers(1, pc)
	frames := runtime.CallersFrames(pc[:n])
	for {
		frame, more := frames.Next()
		fmt.Println(frame.Function, filepath.Base(frame.File), frame.Line)
		if !more || frame.Function == "main.main" {
			break
		}
	}
}

func main() {
	outer()
	walk()
}
