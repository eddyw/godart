package main

import (
	"fmt"
	"reflect"
	"runtime"
)

func run[T ~int](callback func(T) bool) {
	for i := T(0); i < 4; i++ {
		if !callback(i) {
			return
		}
	}
}
func main() {
	for _, function := range []reflect.Value{reflect.ValueOf(run[int])} {
		count := 0
		callback := func(i int) bool { runtime.Gosched(); count++; return true }
		function.Call([]reflect.Value{reflect.ValueOf(callback)})
		if count != 4 {
			panic(fmt.Sprintf("generic callback count: %d", count))
		}
		fmt.Println(count)
	}
}
