package main

import (
	"fmt"
	"maps"
	"slices"
)

func main() {
	s := []int{3, 1, 2}

	for i, v := range slices.All(s) {
		fmt.Print(i, v, " ")
	}
	fmt.Println()

	for v := range slices.Values(s) {
		fmt.Print(v, " ")
	}
	fmt.Println()

	for i, v := range slices.Backward(s) {
		fmt.Print(i, v, " ")
	}
	fmt.Println()

	fmt.Println(slices.Collect(slices.Values(s)))
	fmt.Println(slices.Sorted(slices.Values(s)))
	fmt.Println(slices.AppendSeq([]int{9}, slices.Values(s)))

	for part := range slices.Chunk(s, 2) {
		fmt.Print(part, " ")
	}
	fmt.Println()

	m := map[string]int{"b": 2, "a": 1}
	fmt.Println(slices.Sorted(maps.Keys(m)))
	fmt.Println(slices.Sorted(maps.Values(m)))

	got := map[string]int{}
	maps.Insert(got, maps.All(m))
	fmt.Println(len(got), got["a"])
	fmt.Println(len(maps.Collect(maps.All(m))))

	// A break stops an iterator of the library.
	for v := range slices.Values([]int{1, 2, 3, 4}) {
		if v == 3 {
			break
		}
		fmt.Print(v, " ")
	}
	fmt.Println()

	fmt.Println(slices.SortedFunc(slices.Values(s), func(a, b int) int { return b - a }))
}
