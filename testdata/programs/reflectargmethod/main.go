package main

import (
	"fmt"
	"reflect"
	"runtime"
)

type Runner[T ~int] struct{}

func (Runner[T]) Run(callback func(T) bool) {
	for i := T(0); i < 4; i++ {
		if !callback(i) {
			return
		}
	}
}
func (Runner[T]) Factory() func() T { return func() T { runtime.Gosched(); return T(47) } }

func main() {
	callback := reflect.ValueOf(Runner[int]{}).MethodByName("Factory").Call(nil)[0].Interface().(func() int)
	value := callback()
	if value != 47 {
		panic(fmt.Sprintf("returned callback value: %d", value))
	}

	for _, function := range []reflect.Value{reflect.ValueOf(Runner[int]{}).MethodByName("Run")} {
		count := 0
		callback := func(i int) bool { runtime.Gosched(); count++; return true }
		function.Call([]reflect.Value{reflect.ValueOf(callback)})
		if count != 4 {
			panic(fmt.Sprintf("generic callback count: %d", count))
		}
		fmt.Println(count)
	}
}
