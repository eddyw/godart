package main

import "fmt"

func f() error { return nil }

func shapes() {
	type point struct{ X, Y int }
	type ints map[string]int
	type list []int
	type reader interface{ Read() string }

	p := point{1, 2}
	var a any = p
	fmt.Println(a, p)
	if q, ok := a.(point); ok {
		fmt.Println("assert", q.X+q.Y)
	}

	m := ints{"a": 1}
	m["b"] = 2
	fmt.Println(len(m), m["a"], m["b"], m)

	l := list{3, 4}
	l = append(l, 5)
	fmt.Println(l, len(l), l[2])

	var r reader
	fmt.Println(r == nil)

	var v any = m
	if got, ok := v.(ints); ok {
		fmt.Println("map assert", len(got))
	}
	var w any = l
	if got, ok := w.(list); ok {
		fmt.Println("slice assert", got[0])
	}
	fmt.Printf("%T %T\n", p, l)
}

func nils() {
	fmt.Println(f(), error(nil), f() == error(nil))
	var e error
	var a any = error(nil)
	var b any = e
	fmt.Println(a == b, a == nil, b == nil)
	fmt.Printf("%v %v\n", error(nil), e)

	var s []int = nil
	fmt.Println([]int(nil) == nil, s == nil)
	var mp map[string]int
	fmt.Println(map[string]int(nil) == nil, mp == nil)
}

func main() {
	shapes()
	nils()
}
