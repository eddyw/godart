package main

import (
	"fmt"
	"runtime"
)

func indirect() any {
	runtime.Gosched()
	return recover()
}

func catch(result *any) {
	runtime.Gosched()
	if indirect() != nil {
		panic("indirect recovery")
	}
	*result = recover()
	runtime.Gosched()
	if recover() != nil {
		panic("second recovery")
	}
}

type handler struct{ result *any }

func (h handler) Catch() {
	runtime.Gosched()
	if indirect() != nil {
		panic("method indirect recovery")
	}
	*h.result = recover()
}
func (h *handler) CatchPointer() {
	runtime.Gosched()
	*h.result = recover()
}
func generic[T any](result *T) {
	runtime.Gosched()
	*result = recover().(T)
}

func function() (result any)     { defer catch(&result); panic("function") }
func method() (result any)       { defer (handler{&result}).Catch(); panic("method") }
func pointer() (result any)      { defer (&handler{&result}).CatchPointer(); panic("pointer") }
func parameter() (result string) { defer generic(&result); panic("generic") }
func closure() (result any) {
	defer func() {
		runtime.Gosched()
		helper := func() any { runtime.Gosched(); return recover() }
		if helper() != nil {
			panic("closure indirect recovery")
		}
		result = recover()
	}()
	panic("closure")
}
func main() { fmt.Println(function(), method(), pointer(), parameter(), closure()) }
