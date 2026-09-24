package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
)

type Value struct{ N int }

func (v Value) Method() int { v.N++; return v.N }

type Number int

func (v Number) Method() int { return int(v) + 1 }

type Empty [0]int

func (Empty) Method() int { return 7 }

func report(v reflect.Value) {
	pc := v.Pointer()
	f := runtime.FuncForPC(pc)
	fmt.Println(f != nil)
	if f != nil {
		file, line := f.FileLine(f.Entry())
		fmt.Println(f.Name(), filepath.Base(file), line)
	}
}

func main() {
	for _, value := range []any{Value{13}, Number(20), Empty{}} {
		typ := reflect.PointerTo(reflect.TypeOf(value))
		method, _ := typ.MethodByName("Method")
		report(method.Func)
		pointer := reflect.New(typ.Elem())
		pointer.Elem().Set(reflect.ValueOf(value))
		fmt.Println(method.Func.Call([]reflect.Value{pointer})[0].Int())
		fmt.Println(pointer.Elem().Interface())
		func() {
			defer func() { fmt.Println(recover()) }()
			method.Func.Call([]reflect.Value{reflect.Zero(typ)})
		}()
	}
	report(reflect.ValueOf(error.Error))
	fmt.Println(reflect.ValueOf(error.Error).Call([]reflect.Value{reflect.ValueOf(errors.New("message"))})[0].String())
}
