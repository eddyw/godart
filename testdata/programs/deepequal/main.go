package main

import (
	"fmt"
	"reflect"
)

type point struct {
	X, Y int
	name string
}

type holder struct {
	P     *point
	Items []int
	Table map[string][]int
	Any   any
}

func main() {
	fmt.Println("ints:", reflect.DeepEqual(1, 1), reflect.DeepEqual(1, 2))
	fmt.Println("types:", reflect.DeepEqual(1, int64(1)), reflect.DeepEqual("a", "a"))

	a := []int{1, 2, 3}
	b := []int{1, 2, 3}
	c := []int{1, 2, 4}
	fmt.Println("slices:", reflect.DeepEqual(a, b), reflect.DeepEqual(a, c))
	fmt.Println("nil slice:", reflect.DeepEqual([]int(nil), []int{}), reflect.DeepEqual([]int(nil), []int(nil)))

	m1 := map[string][]int{"a": {1}, "b": {2, 3}}
	m2 := map[string][]int{"b": {2, 3}, "a": {1}}
	m3 := map[string][]int{"a": {1}, "b": {2, 4}}
	fmt.Println("maps:", reflect.DeepEqual(m1, m2), reflect.DeepEqual(m1, m3))

	p1 := point{X: 1, Y: 2, name: "a"}
	p2 := point{X: 1, Y: 2, name: "a"}
	p3 := point{X: 1, Y: 2, name: "b"}
	fmt.Println("structs:", reflect.DeepEqual(p1, p2), reflect.DeepEqual(p1, p3))

	fmt.Println("pointers:", reflect.DeepEqual(&p1, &p2), reflect.DeepEqual(&p1, &p3))

	arr1 := [3]string{"a", "b", "c"}
	arr2 := [3]string{"a", "b", "c"}
	arr3 := [3]string{"a", "b", "d"}
	fmt.Println("arrays:", reflect.DeepEqual(arr1, arr2), reflect.DeepEqual(arr1, arr3))

	h1 := holder{P: &p1, Items: a, Table: m1, Any: p1}
	h2 := holder{P: &p2, Items: b, Table: m2, Any: p2}
	h3 := holder{P: &p3, Items: c, Table: m3, Any: p3}
	fmt.Println("nested:", reflect.DeepEqual(h1, h2), reflect.DeepEqual(h1, h3))

	n1 := 5
	n2 := 5
	n3 := 6
	fmt.Println("pointer to int:", reflect.DeepEqual(&n1, &n2), reflect.DeepEqual(&n1, &n3))

	var e1 error
	var e2 error
	fmt.Println("nil:", reflect.DeepEqual(e1, e2), reflect.DeepEqual(nil, nil))

	fmt.Println("type of:", reflect.TypeOf(p1).String(), reflect.TypeOf(a).Kind())
	fmt.Println("comparable:", reflect.TypeOf(1).Comparable(), reflect.TypeOf(a).Comparable())
}
