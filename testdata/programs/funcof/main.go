// The program builds a function type at run time with reflect.FuncOf, and it
// runs a value of that type.
package main

import (
	"fmt"
	"reflect"
)

type pair struct {
	A int
	B string
}

func main() {
	f := func(a int, b string) bool { return a > 0 && b != "" }
	ti := reflect.TypeOf(0)
	ts := reflect.TypeOf("")
	tb := reflect.TypeOf(true)

	t := reflect.FuncOf([]reflect.Type{ti, ts}, []reflect.Type{tb}, false)
	fmt.Println(t, t.Kind(), t.NumIn(), t.NumOut(), t.IsVariadic())
	fmt.Println(t.In(0), t.In(1), t.Out(0))
	fmt.Println("equal to the type of the value:", t == reflect.TypeOf(f))

	v := reflect.MakeFunc(t, func(in []reflect.Value) []reflect.Value {
		return []reflect.Value{reflect.ValueOf(in[0].Int() > 2)}
	})
	g := v.Interface().(func(int, string) bool)
	fmt.Println("the call gives:", g(5, "x"), g(1, "x"))

	fmt.Println(reflect.FuncOf(nil, nil, false))
	fmt.Println(reflect.FuncOf(nil, []reflect.Type{ti, ts}, false))
	fmt.Println(reflect.FuncOf([]reflect.Type{reflect.SliceOf(ti)}, nil, true))
	fmt.Println(reflect.FuncOf([]reflect.Type{reflect.TypeOf(pair{})},
		[]reflect.Type{reflect.PointerTo(reflect.TypeOf(pair{}))}, false))
}
