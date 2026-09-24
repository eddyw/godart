package main

import (
	"fmt"
	"reflect"
	"runtime"
)

type A struct{}
type B struct{}
type I interface{ F() }
type J interface{ G() }

func (*A) F() {}
func (*B) F() {}

func generic[T any](v any) T { return v.(T) }

func probe(f func()) {
	defer func() {
		v := recover()
		e, ok := v.(*runtime.TypeAssertionError)
		_, marker := v.(runtime.Error)
		fmt.Printf("%T %v %v %v\n", v, ok, marker, v == v)
		if ok {
			copy := *e
			fmt.Println(e.Error(), copy.Error())
			err := reflect.ValueOf(e).MethodByName("Error").Call(nil)
			fmt.Println(err[0].String())
		}
	}()
	f()
}

func main() {
	var a any = (*A)(nil)
	var i I = (*A)(nil)
	var nilAny any
	var nilI I
	for _, f := range []func(){
		func() { _ = a.(*B) },
		func() { _ = i.(*B) },
		func() { _ = nilAny.(*B) },
		func() { _ = nilI.(*B) },
		func() { _ = a.(J) },
		func() { _ = i.(J) },
		func() { _ = nilAny.(J) },
		func() { _ = nilI.(J) },
		func() { generic[*B](a) },
		func() { generic[*B](nilAny) },
		func() { generic[J](a) },
		func() { generic[J](nilAny) },
	} {
		probe(f)
	}
	fmt.Println(generic[*A](a) == nil, generic[I](a).(*A) == nil)
}
