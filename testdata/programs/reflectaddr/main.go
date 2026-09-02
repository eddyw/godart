package main

import (
	"fmt"
	"reflect"
)

type pair struct{ A, B int }

type row struct {
	N   int
	M   int
	S   string
	F   float64
	T   bool
	P   pair
	Arr [2]int
	E   error
}

func main() {
	r := row{N: 3, M: 4, S: "go", F: 1.5, T: true, P: pair{A: 4, B: 5}}
	v := reflect.ValueOf(&r).Elem()

	// The address of a field carries the type of the field, thus a type
	// assertion of the result reads.
	pn := v.Field(0).Addr().Interface().(*int)
	*pn = 7
	ps := v.Field(2).Addr().Interface().(*string)
	*ps = "dart"
	pf := v.Field(3).Addr().Interface().(*float64)
	*pf = 2.5
	pt := v.Field(4).Addr().Interface().(*bool)
	*pt = false
	pp := v.Field(5).Addr().Interface().(*pair)
	pp.B = 9
	pa := v.Field(6).Addr().Interface().(*[2]int)
	pa[1] = 8
	pe := v.Field(7).Addr().Interface().(*error)
	*pe = fmt.Errorf("boom")
	fmt.Println(r.N, r.S, r.F, r.T, r.P, r.Arr, r.E)

	// The address of an element of a slice and of an array reads too.
	s := []int{10, 11}
	e := reflect.ValueOf(s).Index(1).Addr().Interface().(*int)
	*e = 12
	fmt.Println(s)
	a := [2]string{"x", "y"}
	q := reflect.ValueOf(&a).Elem().Index(0).Addr().Interface().(*string)
	*q = "z"
	fmt.Println(a)

	// The address of the value behind a pointer gives that pointer again.
	n := 20
	pv := reflect.ValueOf(&n).Elem().Addr().Interface().(*int)
	*pv = 21
	fmt.Println(n, pv == &n)

	// 2 reads of the address of 1 field give equal pointers, and 2 fields
	// give 2 pointers.
	fmt.Println(v.Field(0).Addr().Interface().(*int) ==
		v.Field(0).Addr().Interface().(*int))
	fmt.Println(v.Field(0).Addr().Interface().(*int) ==
		v.Field(1).Addr().Interface().(*int))

	// The type of the pointer follows the field, and Elem reads the value
	// again.
	fmt.Println(v.Field(0).Addr().Type(), reflect.TypeOf(pn))
	fmt.Println(v.Field(0).Addr().Elem().Interface())
}
