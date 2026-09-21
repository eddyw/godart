package main

import (
	"example.com/funcpc/helper"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
)

func named(n int) int { return n + 1 }

var global = func(n int) int { return n + 4 }

//go:noinline
func nested(n int) func() func(int) int {
	return func() func(int) int { return func(x int) int { return n + x } }
}

//go:noinline
func closure(n int) func(int) int {
	return func(x int) int { return x + n }
}

//go:noinline
func blocking(n int) func() int {
	return func() int { runtime.Gosched(); return n }
}

func main() {
	first, second := closure(2), closure(3)
	firstPC := reflect.ValueOf(first).Pointer()
	secondPC := reflect.ValueOf(second).Pointer()
	fn := runtime.FuncForPC(firstPC)
	if fn == nil {
		panic("missing closure metadata")
	}
	file, line := fn.FileLine(firstPC)
	fmt.Println("closure", fn.Name(), filepath.Base(file), line > 0, fn.Entry() == firstPC, firstPC == secondPC)
	fmt.Println("captures", first(10), second(10))
	for _, value := range []any{global, nested(5)(), blocking(6)} {
		pc := reflect.ValueOf(value).Pointer()
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			panic("missing literal metadata")
		}
		file, line := fn.FileLine(pc)
		fmt.Println(fn.Name(), filepath.Base(file), line > 0)
	}
	fmt.Println("blocking call", blocking(6)())
	runtime.Caller(0)
	for _, bad := range []uintptr{0, ^uintptr(0), ^uintptr(0) >> 1, uintptr(1) << 63} {
		f := runtime.FuncForPC(bad)
		fmt.Printf("%t %q\n", f == nil, f.Name())
	}
}

func init() {
	for _, value := range []any{named, helper.Named} {
		pc := reflect.ValueOf(value).Pointer()
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			panic("missing function metadata")
		}
		file, line := fn.FileLine(pc)
		fmt.Println(fn.Name(), filepath.Base(file), line > 0, fn.Entry() == pc, fn == runtime.FuncForPC(pc))
	}
}
