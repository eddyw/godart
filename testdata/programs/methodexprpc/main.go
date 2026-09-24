package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
)

type Number int

func (Number) Read() int { return 3 }
func (*Number) Write()   {}

type Inner struct{}

func (Inner) Value() int { return 4 }
func (*Inner) Pointer()  {}

type Outer struct{ Inner }

func compare(t reflect.Type, name string, expression any) {
	method, ok := t.MethodByName(name)
	if !ok {
		panic(name)
	}
	direct := reflect.ValueOf(expression).Pointer()
	first := method.Func.Pointer()
	again, _ := t.MethodByName(name)
	fmt.Println(t, name, first == direct, again.Func.Pointer() == direct)
	for _, pc := range []uintptr{direct, first} {
		function := runtime.FuncForPC(pc)
		file, line := function.FileLine(function.Entry())
		fmt.Println(function.Name(), filepath.Base(file), line)
	}
}

func main() {
	compare(reflect.TypeOf(Number(0)), "Read", Number.Read)
	compare(reflect.TypeOf((*Number)(nil)), "Read", (*Number).Read)
	compare(reflect.TypeOf((*Number)(nil)), "Write", (*Number).Write)
	compare(reflect.TypeOf(Inner{}), "Value", Inner.Value)
	compare(reflect.TypeOf((*Inner)(nil)), "Value", (*Inner).Value)
	compare(reflect.TypeOf((*Inner)(nil)), "Pointer", (*Inner).Pointer)
	compare(reflect.TypeOf(Outer{}), "Value", Outer.Value)
	compare(reflect.TypeOf((*Outer)(nil)), "Value", (*Outer).Value)
	compare(reflect.TypeOf((*Outer)(nil)), "Pointer", (*Outer).Pointer)
	var f = error.Error
	fmt.Println(f(errors.New("error method")))
}
