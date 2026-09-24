package main

import "fmt"

func probe(f func()) {
	defer func() { fmt.Println(recover()) }()
	f()
}

func main() {
	for _, v := range []any{[0][]int{}, [0]map[int]int{}, [0]func(){}, struct{ X [0][]int }{}, struct {
		X int
		Y []int
	}{}} {
		fmt.Printf("%T\n", v)
		probe(func() { fmt.Println(v == v) })
		probe(func() { m := map[any]int{}; m[v] = 1; fmt.Println(len(m)) })
		probe(func() { m := map[any]int{}; fmt.Println(m[v]) })
		probe(func() { var m map[any]int; delete(m, v) })
	}
	a := any(struct {
		X int
		Y []int
	}{X: 1})
	b := any(struct {
		X int
		Y []int
	}{X: 2})
	probe(func() { fmt.Println(a == b) })
	probe(func() { fmt.Println(a == nil, a == any([0][]int{})) })
	c := any(struct {
		X int
		Y any
	}{X: 1, Y: []int{}})
	d := any(struct {
		X int
		Y any
	}{X: 2, Y: []int{}})
	probe(func() { fmt.Println(c == d) })
	probe(func() { fmt.Println(c == c) })
	probe(func() { v := any([0]any{}); fmt.Println(v == v, map[any]int{v: 7}[v]) })
}
