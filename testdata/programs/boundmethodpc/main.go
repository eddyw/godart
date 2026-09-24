package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
)

type Number int

func (v Number) Read() int { return int(v) }
func (v *Number) Pointer() int {
	if v == nil {
		return -1
	}
	return int(*v)
}

type Inner struct{ N int }

func (v Inner) Read() int { return v.N }

type Outer struct{ Inner }
type Reader interface{ Read() int }
type Extended interface{ Reader }
type G[T any] struct{ V T }

func (v G[T]) Read() T { return v.V }
func show(name string, f any) {
	v := reflect.ValueOf(f)
	p := v.Pointer()
	fn := runtime.FuncForPC(p)
	fmt.Println(name, fn != nil)
	if fn != nil {
		file, line := fn.FileLine(p)
		fmt.Println(fn.Name(), filepath.Base(file), line, fn.Entry() == p)
	}
}
func bind[T any](v G[T]) (func() T, func() T) {
	v.Read()
	before := func() T { return v.V }
	return before, v.Read
}
func constrained[T Reader](v T) func() int { return v.Read }

func main() {
	var nilPointer *Number
	nilMethod := nilPointer.Pointer
	show("nil pointer", nilMethod)
	fmt.Println(nilMethod())
	func() {
		defer func() { fmt.Println("nil value receiver", recover()) }()
		_ = nilPointer.Read
	}()
	func() {
		defer func() { fmt.Println("nil interface", recover()) }()
		var value Reader
		_ = value.Read
	}()
	n := Number(13)
	v, p := n.Read, n.Pointer
	n = 21
	show("value", v)
	show("pointer", p)
	fmt.Println(v(), p())
	show("pointer value", (&n).Read)
	show("embedded", Outer{Inner{4}}.Read)
	var i Reader = n
	show("interface", i.Read)
	var e Extended = n
	show("extended", e.Read)
	var anonymous interface{ Read() int } = n
	show("anonymous", anonymous.Read)
	var err error = errors.New("text")
	show("error", err.Error)
	show("generic", G[int]{7}.Read)
	show("other generic", G[string]{"s"}.Read)
	first, second := G[int]{2}.Read, G[int]{3}.Read
	other := G[string]{"s"}.Read
	fmt.Println(reflect.ValueOf(first).Pointer() == reflect.ValueOf(second).Pointer())
	fmt.Println(reflect.ValueOf(first).Pointer() != reflect.ValueOf(other).Pointer())
	before, bound := bind(G[int]{31})
	show("generic caller closure", before)
	show("generic caller bound", bound)
	fmt.Println(before(), bound())
	constrainedValue := constrained(Number(44))
	show("constrained bound", constrainedValue)
	fmt.Println(constrainedValue())

	reflected := reflect.ValueOf(n).MethodByName("Read").Pointer()
	fmt.Println("reflection", runtime.FuncForPC(reflected).Name(), runtime.FuncForPC(reflected).Entry() == reflected)
	a, b := Number(1).Read, Number(2).Read
	fmt.Println(reflect.ValueOf(a).Pointer() == reflect.ValueOf(b).Pointer())
	fmt.Println(reflect.ValueOf(a).Pointer() != reflect.ValueOf(Number.Read).Pointer())
}
