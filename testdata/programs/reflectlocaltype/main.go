package main

import (
	"fmt"
	"reflect"
)

type Box[T any] struct{ Value T }

func same(name string, actual reflect.Type, expected any) {
	want := reflect.TypeOf(expected)
	fmt.Println(name, actual == want, reflect.TypeOf(reflect.Zero(actual).Interface()) == want)
}

func derived[T any](value T) {
	t := reflect.TypeOf(value)
	same("pointer", reflect.PointerTo(t), (*T)(nil))
	same("slice", reflect.SliceOf(t), []T(nil))
	same("array", reflect.ArrayOf(2, t), [2]T{})
	same("map", reflect.MapOf(reflect.TypeOf(""), t), map[string]T(nil))
	same("channel", reflect.ChanOf(reflect.BothDir, t), chan T(nil))
	same("function", reflect.FuncOf([]reflect.Type{t}, []reflect.Type{t}, false), (func(T) T)(nil))
	same("nested", reflect.SliceOf(reflect.FuncOf([]reflect.Type{t}, nil, false)), ([]func(T))(nil))
}

func first() reflect.Type {
	type Local int
	return reflect.TypeOf((func(Local))(nil))
}

func second() reflect.Type {
	type Local int
	return reflect.TypeOf((func(Local))(nil))
}

func main() {
	type Local int
	t := reflect.TypeOf(Local(0))
	same("direct function", reflect.FuncOf([]reflect.Type{t}, nil, false), (func(Local))(nil))
	same("direct map", reflect.MapOf(t, t), map[Local]Local(nil))
	same("direct nested", reflect.ArrayOf(2, reflect.FuncOf([]reflect.Type{t}, nil, false)), [2]func(Local){})
	derived(Local(0))
	derived(Box[Local]{})
	fmt.Println("distinct", first() != second())
	f := reflect.FuncOf([]reflect.Type{reflect.TypeOf(Local(0))}, nil, false)
	_, ok := reflect.Zero(f).Interface().(func(Local))
	fmt.Println("assertion", ok)
}
