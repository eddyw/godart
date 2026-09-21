package main

import (
	"fmt"
	"reflect"
)

type Point struct{ N int }
type Holder[A, B any] struct {
	F func(A, *B) A
	G func(A) *B
}

func main() {
	calls := 0
	h := Holder[int, Point]{
		F: func(a int, b *Point) int { calls++; return a + b.N },
		G: func(a int) *Point { return &Point{N: a} },
	}
	v := reflect.ValueOf(&h).Elem()
	f := v.Field(0).Interface().(func(int, *Point) int)
	if f(2, &Point{N: 3}) != 5 || calls != 1 {
		panic("field getter")
	}
	v.Field(0).Set(reflect.ValueOf(func(a int, b *Point) int { return a * b.N }))
	if h.F(4, &Point{N: 5}) != 20 {
		panic("field setter")
	}
	g := v.Field(1).Interface().(func(int) *Point)
	if g(6).N != 6 {
		panic("pointer result")
	}
	v.Field(1).Set(reflect.ValueOf(func(a int) *Point { return &Point{N: a + 1} }))
	if h.G(7).N != 8 {
		panic("pointer result setter")
	}
	v.Field(0).Set(reflect.Zero(v.Field(0).Type()))
	if h.F != nil || !v.Field(0).IsNil() {
		panic("nil function")
	}
	scalar := Holder[string, int]{F: func(a string, b *int) string { *b++; return a }, G: func(a string) *int { n := len(a); return &n }}
	sv := reflect.ValueOf(scalar)
	sf := sv.Field(0).Interface().(func(string, *int) string)
	n := 10
	if sf("value", &n) != "value" || n != 11 {
		panic("boxed pointer argument")
	}
	sg := sv.Field(1).Interface().(func(string) *int)
	if *sg("value") != 5 {
		panic("boxed pointer result")
	}
	fmt.Println("ok")
}
