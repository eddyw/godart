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

	var nilPoints []point
	fmt.Println(reflect.Append(reflect.ValueOf(nilPoints)).IsNil())
	points := make([]point, 2, 4)
	second := &points[1]
	added := reflect.Append(reflect.ValueOf(points[:1]), reflect.ValueOf(point{3, 4})).Interface().([]point)
	fmt.Println(second == &added[1], *second, len(points), len(added))
	source := []point{{5, 6}}
	more := reflect.AppendSlice(reflect.ValueOf(added), reflect.ValueOf(source)).Interface().([]point)
	source[0].X = 99
	fmt.Println(more[2], source[0])
	old := &more[0]
	grown := reflect.AppendSlice(reflect.ValueOf(more), reflect.ValueOf([]point{{7, 8}, {9, 10}})).Interface().([]point)
	grown[0].X = 77
	fmt.Println(*old, grown[0])
	left, right := &point{1, 2}, &point{1, 2}
	fmt.Println(reflect.ValueOf(left).Equal(reflect.ValueOf(right)), reflect.ValueOf(left).Equal(reflect.ValueOf(left)))
	fmt.Println(reflect.ValueOf(*left).Field(0).CanAddr(), reflect.ValueOf(left).Elem().Field(0).CanAddr())
	fmt.Println(reflect.ValueOf([1]int{1}).Index(0).CanAddr(), reflect.ValueOf([]int{1}).Index(0).CanAddr())
}
