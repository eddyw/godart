package main

import (
	"example.com/privateforward/left"
	"example.com/privateforward/right"
	"fmt"
)

type Pointer struct{ *left.S }
type Value struct{ left.V }
type Scalar struct{ left.Number }
type Hidden struct{ left.Outer }
type Forward struct{ left.I }
type Generic[T any] struct{ left.Box[T] }
type Local interface{ get() int }

func main() {
	p := Pointer{&left.S{Value: 4}}
	if left.Call(p) != 5 || p.Value != 5 {
		panic("pointer receiver")
	}
	if left.Call(Pointer{}) != -1 {
		panic("nil receiver")
	}
	v := Value{left.V{Value: 6}}
	if left.Call(v) != 7 || v.Value != 6 {
		panic("value receiver")
	}
	if left.Call(Scalar{8}) != 8 {
		panic("scalar receiver")
	}
	if left.Call(Hidden{left.Make()}) != 9 {
		panic("private embedded field")
	}
	var x left.I = p
	for i := 0; i < 1000; i++ {
		x = Forward{x}
	}
	if left.Call(x) != 6 || p.Value != 6 {
		panic("interface forwarding")
	}
	if _, ok := any(p).(Local); ok {
		panic("local method identity")
	}
	if right.Implements(p) {
		panic("foreign method identity")
	}
	if left.Read[int](Generic[int]{left.Box[int]{Value: 10}}) != 10 {
		panic("generic receiver")
	}
	fmt.Println("ok")
}
