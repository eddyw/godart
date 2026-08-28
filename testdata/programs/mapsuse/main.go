package main

import (
	"fmt"
	"maps"
	"slices"
)

type counts map[string]int

func isNilMap[M ~map[K]V, K comparable, V any](m M) bool { return m == nil }

func isNilSlice[S ~[]E, E any](s S) bool { return s == nil }

func main() {
	a := map[string]int{"x": 1, "y": 2}
	b := map[string]int{"x": 1, "y": 2}
	fmt.Println(maps.Equal(a, b), maps.Equal(a, map[string]int{"x": 1}))

	c := maps.Clone(a)
	c["x"] = 99
	fmt.Println(a["x"], c["x"], len(c))

	dst := map[string]int{}
	maps.Copy(dst, a)
	fmt.Println(len(dst), dst["y"])

	maps.DeleteFunc(dst, func(k string, v int) bool { return v > 1 })
	fmt.Println(len(dst), dst["x"])

	fmt.Println(maps.EqualFunc(a, b, func(p, q int) bool { return p == q }))

	var named counts = map[string]int{"a": 1}
	fmt.Println(maps.Clone(named)["a"], maps.Clone(counts(nil)) == nil)

	// A nil test through a type parameter reads the nil field of the value.
	var nilMap map[string]int
	var nilSlice []int
	fmt.Println(isNilMap(nilMap), isNilMap(map[string]int{}))
	fmt.Println(isNilSlice(nilSlice), isNilSlice([]int{}))

	// The order of a map is not stable, thus the keys need a sort.
	keys := make([]string, 0, len(a))
	for k := range a {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	fmt.Println(keys)
}
