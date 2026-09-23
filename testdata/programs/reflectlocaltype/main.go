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

func localGeneric[A, B any](a A, b B) {
	type Local struct {
		A A
		B B
	}
	value := Local{a, b}
	typ := reflect.TypeOf(value)
	same("generic local pointer", reflect.PointerTo(typ), (*Local)(nil))
	same("generic local slice", reflect.SliceOf(typ), []Local(nil))
	same("generic local channel", reflect.ChanOf(reflect.BothDir, typ), chan Local(nil))
	same("generic local function", reflect.FuncOf([]reflect.Type{typ}, []reflect.Type{typ}, false), (func(Local) Local)(nil))
	pointer := reflect.New(typ).Interface().(*Local)
	*pointer = value
	fmt.Println("generic local value", pointer.A, pointer.B)
	var ch chan Local
	fmt.Println("generic local nil channel", ch == nil)
	ch = make(chan Local, 1)
	ch <- value
	fmt.Println("generic local channel value", <-ch)
	close(ch)
	zero, ok := <-ch
	fmt.Println("generic local channel zero", zero, ok)
}

func main() {
	localGeneric(7, "text")
	localGeneric("value", false)
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
