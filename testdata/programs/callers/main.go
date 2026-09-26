package main

import (
	"fmt"
	"path/filepath"
	"reflect"
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
	identity()
	callSites()
}

//go:noinline
func identity() {
	first, file, line, _ := runtime.Caller(0)
	second, _, next, _ := runtime.Caller(0)
	f := runtime.FuncForPC(first)
	g := runtime.FuncForPC(second)
	entry := reflect.ValueOf(identity).Pointer()
	entryFile, entryLine := f.FileLine(f.Entry())
	_, firstLine := f.FileLine(first)
	_, secondLine := f.FileLine(second)
	fmt.Println("identity", f == g, f == runtime.FuncForPC(entry), f.Entry() == entry)
	fmt.Println("positions", entryFile == file, entryLine < line, firstLine == line, secondLine == next, firstLine != secondLine)
	frames := runtime.CallersFrames([]uintptr{first, second})
	a, more := frames.Next()
	b, last := frames.Next()
	fmt.Println("entries", a.Entry == entry, b.Entry == entry, a.Func == b.Func, more, last)
}

func sourceSite(label string, value int) int {
	pc, _, line, _ := runtime.Caller(1)
	fmt.Println(label, runtime.FuncForPC(pc).Name(), line)
	return value
}

func callSites() {
	sourceSite("outer",
		sourceSite("inner", 1))
	for i := 0; sourceSite("condition", i) < 2; i = sourceSite("post", i+1) {
		sourceSite("body", i)
	}
}
