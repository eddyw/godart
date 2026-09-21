package main

import (
	"fmt"
	"reflect"
)

type step interface{ Next() int }
type wrapper struct{ step }
type outer struct{ wrapper }
type reader struct{ ready <-chan int }

func (r reader) Next() int { return <-r.ready }

type number int

func (n number) Next() int { return int(n) }

type optional struct{ value int }

func (p *optional) Next() int {
	if p == nil {
		return -1
	}
	return p.value
}

type transform[T any] interface{ Apply(T, func(T) T) T }
type genericWrapper[T any] struct{ transform[T] }
type transformValue[T any] struct{}

func (transformValue[T]) Apply(v T, f func(T) T) T { return f(v) }

func panics(s step) (result bool) {
	defer func() { result = recover() != nil }()
	s.Next()
	return
}

func main() {
	ready := make(chan int)
	go func() {
		ready <- 7
		ready <- 11
	}()
	var first step = wrapper{reader{ready}}
	fmt.Println(first.Next())
	var second step = outer{wrapper{reader{ready}}}
	fmt.Println(second.Next())

	w := wrapper{number(3)}
	var value step = w
	var pointer step = &w
	method := pointer.Next
	w.step = number(5)
	fmt.Println(value.Next(), pointer.Next(), method())
	fmt.Println(reflect.ValueOf(pointer).MethodByName("Next").Call(nil)[0].Int())
	var anonymous step = struct{ step }{pointer}
	fmt.Println(anonymous.Next())

	var typedNil *optional
	var nullable step = wrapper{typedNil}
	fmt.Println(nullable.Next())
	var empty *wrapper
	var nested struct{ *wrapper }
	fmt.Println(panics(wrapper{}), panics(empty), panics(nested))

	var generic transform[int] = genericWrapper[int]{transformValue[int]{}}
	for range 100 {
		generic = genericWrapper[int]{generic}
	}
	fmt.Println(generic.Apply(9, func(v int) int { return v + 4 }))
}
