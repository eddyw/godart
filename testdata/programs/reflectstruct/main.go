// The program reads a descriptor of a struct type and of an array type. Each 1
// must name the Dart type of a value, thus reflect.New and Value.Addr each give
// a pointer that a type assertion reads. The witness is fixedbugs/issue37716 of
// the test suite of Go.
package main

import (
	"fmt"
	"reflect"
)

// K needs a hash of the compiler in Go, thus the witness reads it.
type K struct {
	a, b int32
	c    float64
}

type M map[K]bool

type box struct {
	Name string
	Nums [3]int
}

func main() {
	k := K{a: 1, b: 2, c: 3}

	// issue37716: a map that MapOf builds fits a location of a named map type.
	m := reflect.MakeMap(reflect.MapOf(reflect.TypeOf(K{}), reflect.TypeOf(true)))
	m.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(true))
	var x M
	reflect.ValueOf(&x).Elem().Set(m)
	fmt.Println(x[k], len(x))

	// New of a struct type gives a pointer that a type assertion reads.
	p := reflect.New(reflect.TypeOf(box{}))
	p.Elem().Field(0).SetString("one")
	fmt.Println(p.Interface().(*box).Name)

	// A slice of a struct type that SliceOf builds reads as such a slice.
	bs := reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(box{})), 1, 1)
	bs.Index(0).Field(0).SetString("two")
	fmt.Println(bs.Interface().([]box)[0].Name)

	// The address of a field of an array type reads as such a pointer.
	var b box
	fa := reflect.ValueOf(&b).Elem().Field(1).Addr()
	fmt.Println(fa.Interface().(*[3]int) != nil)
	fa.Elem().Index(2).SetInt(9)
	fmt.Println(b.Nums)

	// A pointer to a struct that New builds keeps each method.
	pv := reflect.New(reflect.TypeOf(counter{}))
	pv.Interface().(*counter).Add(4)
	fmt.Println(pv.Elem().Interface().(counter).N)
}

type counter struct{ N int }

func (c *counter) Add(n int) { c.N += n }
