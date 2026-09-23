package main

import (
	"fmt"
	"reflect"
)

type Record = struct{ X int }

func same(actual reflect.Type, value any) {
	fmt.Println(actual == reflect.TypeOf(value))
}

func main() {
	record := reflect.TypeOf(Record{})
	integer := reflect.TypeOf(0)
	zero := reflect.FuncOf([]reflect.Type{record}, nil, false)
	same(zero, (func(Record))(nil))
	value := 0
	f0 := reflect.MakeFunc(zero, func(args []reflect.Value) []reflect.Value {
		value = int(args[0].Field(0).Int())
		return nil
	}).Interface().(func(Record))
	f0(Record{5})
	fmt.Println(value)

	one := reflect.FuncOf(nil, []reflect.Type{record}, false)
	f1, ok := reflect.MakeFunc(one, func([]reflect.Value) []reflect.Value {
		return []reflect.Value{reflect.ValueOf(Record{7})}
	}).Interface().(func() Record)
	fmt.Println(ok)
	if ok {
		fmt.Println(f1().X)
	}

	two := reflect.FuncOf([]reflect.Type{record}, []reflect.Type{record, integer}, false)
	same(two, (func(Record) (Record, int))(nil))
	f2 := reflect.MakeFunc(two, func(args []reflect.Value) []reflect.Value {
		return []reflect.Value{args[0], reflect.ValueOf(8)}
	}).Interface().(func(Record) (Record, int))
	x, n := f2(Record{6})
	fmt.Println(x.X, n)

	variadic := reflect.FuncOf([]reflect.Type{reflect.SliceOf(record)}, []reflect.Type{integer}, true)
	same(variadic, (func(...Record) int)(nil))
	fv := reflect.MakeFunc(variadic, func(args []reflect.Value) []reflect.Value {
		return []reflect.Value{reflect.ValueOf(args[0].Len())}
	}).Interface().(func(...Record) int)
	fmt.Println(fv(), fv(Record{1}, Record{2}))

	same(reflect.SliceOf(two), ([]func(Record) (Record, int))(nil))
	same(reflect.StructOf([]reflect.StructField{{Name: "F", Type: zero}}), struct{ F func(Record) }{})
	same(reflect.StructOf([]reflect.StructField{{Name: "F", Type: reflect.TypeOf((func(int) int)(nil))}}), struct{ F func(int) int }{})
}
