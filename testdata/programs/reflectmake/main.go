package main

import (
	"fmt"
	"reflect"
)

type point struct{ X, Y int }

func main() {
	it := reflect.TypeOf(0)
	st := reflect.TypeOf("")

	// The type constructors give a Type that reads.
	fmt.Println(reflect.SliceOf(it), reflect.SliceOf(it).Kind(), reflect.SliceOf(it).Elem())
	fmt.Println(reflect.ArrayOf(3, st), reflect.ArrayOf(3, st).Len())
	fmt.Println(reflect.MapOf(st, it), reflect.MapOf(st, it).Key(), reflect.MapOf(st, it).Elem())
	fmt.Println(reflect.PointerTo(it), reflect.PointerTo(it).Elem())
	fmt.Println(reflect.PointerTo(reflect.TypeOf(point{})))

	// TypeFor names a type with no value.
	fmt.Println(reflect.TypeFor[int](), reflect.TypeFor[[]string](), reflect.TypeFor[point]())

	// A type that the program holds keeps its own descriptor, thus MakeSlice
	// of it works.
	var sample []int
	_ = sample
	sl := reflect.MakeSlice(reflect.SliceOf(it), 2, 4)
	fmt.Println(sl.Len(), sl.Cap(), sl.Interface())

	// Append and AppendSlice.
	a := reflect.ValueOf([]int{1, 2})
	b := reflect.Append(a, reflect.ValueOf(3), reflect.ValueOf(4))
	fmt.Println(b.Interface(), b.Len(), b.Cap() >= 4)
	c := reflect.AppendSlice(b, reflect.ValueOf([]int{5, 6}))
	fmt.Println(c.Interface())

	// Append to a slice of an interface type.
	d := reflect.Append(reflect.ValueOf([]any{1}), reflect.ValueOf("two"))
	fmt.Println(d.Interface())

	// Copy between 2 slices.
	dst := []int{0, 0, 0}
	n := reflect.Copy(reflect.ValueOf(dst), reflect.ValueOf([]int{7, 8, 9, 10}))
	fmt.Println(n, dst)

	// Copy of a string into a byte slice.
	buf := make([]byte, 4)
	n = reflect.Copy(reflect.ValueOf(buf), reflect.ValueOf("hello"))
	fmt.Println(n, string(buf))
}
