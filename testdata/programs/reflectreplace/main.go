package main

import (
	"fmt"
	"reflect"
	"runtime"
)

func initial(int16, string) (int16, string) { return 0, "" }

func create[F any](callback func([]reflect.Value) []reflect.Value) F {
	return reflect.MakeFunc(reflect.TypeFor[F](), callback).Interface().(F)
}

func main() {
	factories := []func(reflect.Type, func([]reflect.Value) []reflect.Value) reflect.Value{reflect.MakeFunc}
	makeFunction := factories[0]
	f := initial
	made := makeFunction(reflect.TypeOf(f), func(args []reflect.Value) []reflect.Value {
		runtime.Gosched()
		return args
	})
	reflect.ValueOf(&f).Elem().Set(made)
	a, b := f(12, "replaced")
	if a != 12 || b != "replaced" {
		panic("lost replacement result")
	}
	g := made.Interface().(func(int16, string) (int16, string))
	a, b = g(23, "asserted")
	if a != 23 || b != "asserted" {
		panic("lost asserted result")
	}
	higher := makeFunction(reflect.TypeOf((func() func() int)(nil)), func([]reflect.Value) []reflect.Value {
		return []reflect.Value{reflect.ValueOf(func() int { runtime.Gosched(); return 37 })}
	}).Interface().(func() func() int)
	returned := higher()
	if returned() != 37 {
		panic("lost returned function result")
	}
	accept := makeFunction(reflect.TypeOf((func(func() int) int)(nil)), func(args []reflect.Value) []reflect.Value {
		call := args[0].Interface().(func() int)
		return []reflect.Value{reflect.ValueOf(call())}
	}).Interface().(func(func() int) int)
	if accept(func() int { runtime.Gosched(); return 41 }) != 41 {
		panic("lost function argument result")
	}
	generic := create[func() int](func([]reflect.Value) []reflect.Value {
		runtime.Gosched()
		return []reflect.Value{reflect.ValueOf(47)}
	})
	if generic() != 47 {
		panic("lost generic callback result")
	}
	fmt.Println(a, b)
}
