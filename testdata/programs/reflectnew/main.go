package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"reflect"
)

type point struct{ X, Y int }

type count int

func main() {
	// New of a type that is not a struct gives a pointer with the Dart type
	// of the value, thus a type assertion of the result reads.
	pi := reflect.New(reflect.TypeOf(0)).Interface().(*int)
	*pi = 5
	fmt.Println(*pi)
	pu := reflect.New(reflect.TypeOf(uint8(0))).Interface().(*uint8)
	*pu = 200
	fmt.Println(*pu)
	ps := reflect.New(reflect.TypeOf("")).Interface().(*string)
	*ps = "go"
	fmt.Println(*ps)
	pb := reflect.New(reflect.TypeOf(false)).Interface().(*bool)
	*pb = true
	fmt.Println(*pb)
	pf := reflect.New(reflect.TypeOf(1.5)).Interface().(*float64)
	*pf = 2.5
	fmt.Println(*pf)
	pc := reflect.New(reflect.TypeOf(complex(1.0, 2.0))).Interface().(*complex128)
	*pc = complex(3, 4)
	fmt.Println(*pc)
	pn := reflect.New(reflect.TypeOf(count(0))).Interface().(*count)
	*pn = 9
	fmt.Println(*pn)

	// A struct, an array, and an interface each read too.
	pp := reflect.New(reflect.TypeOf(point{})).Interface().(*point)
	pp.X = 1
	fmt.Println(*pp)
	pa := reflect.New(reflect.TypeOf([3]int{})).Interface().(*[3]int)
	pa[2] = 6
	fmt.Println(*pa)
	pe := reflect.New(reflect.TypeOf((*error)(nil)).Elem()).Interface().(*error)
	*pe = fmt.Errorf("boom")
	fmt.Println(*pe)

	// A slice, a map, a channel, a function, and a pointer each name the Dart
	// type of the element. The description of the type carries that name since
	// 2026-09-03, thus these 5 read too.
	pls := reflect.New(reflect.TypeOf([]int(nil))).Interface().(*[]int)
	*pls = append(*pls, 7)
	fmt.Println(*pls)
	pm := reflect.New(reflect.TypeOf(map[string]int(nil))).Interface().(*map[string]int)
	*pm = map[string]int{"a": 1}
	fmt.Println(*pm)
	pch := reflect.New(reflect.TypeOf(make(chan int))).Interface().(*chan int)
	*pch = make(chan int, 1)
	*pch <- 4
	fmt.Println(<-*pch)
	pfn := reflect.New(reflect.TypeOf((func(int) int)(nil))).Interface().(*func(int) int)
	*pfn = func(x int) int { return x + 1 }
	fmt.Println((*pfn)(1))
	pptr := reflect.New(reflect.TypeOf((*int)(nil))).Interface().(**int)
	inner := 11
	*pptr = &inner
	fmt.Println(**pptr)

	// A write of such a pointer into a typed pointer slot reads.
	var slot *int
	reflect.ValueOf(&slot).Elem().Set(reflect.New(reflect.TypeOf(0)))
	*slot = 8
	fmt.Println(*slot)

	// The value behind the fresh pointer reads and writes through reflect.
	fresh := reflect.New(reflect.TypeOf(0))
	fresh.Elem().SetInt(3)
	fmt.Println(fresh.Elem().Int(), fresh.Type())

	// A gob stream that carries a pointer to a pointer decodes. The decoder
	// allocates the inner pointer with reflect.New.
	var buf bytes.Buffer
	n := 42
	p := &n
	if err := gob.NewEncoder(&buf).Encode(&p); err != nil {
		fmt.Println("encode:", err)
		return
	}
	var out *int
	if err := gob.NewDecoder(&buf).Decode(&out); err != nil {
		fmt.Println("decode:", err)
		return
	}
	fmt.Println(*out)
}
