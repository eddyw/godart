package main

import (
	"fmt"
	"slices"
	"strings"
)

type ids []int

func main() {
	s := []int{5, 2, 8, 1, 9, 3}
	slices.Sort(s)
	fmt.Println(s, slices.IsSorted(s))

	fmt.Println(slices.Index(s, 8), slices.Contains(s, 4))
	fmt.Println(slices.Min(s), slices.Max(s))

	i, ok := slices.BinarySearch(s, 8)
	fmt.Println(i, ok)
	i, ok = slices.BinarySearch(s, 4)
	fmt.Println(i, ok)

	fmt.Println(slices.Equal(s, s), slices.Equal(s, []int{1}))
	fmt.Println(slices.Compare([]int{1, 2}, []int{1, 3}))

	c := slices.Clone(s)
	c[0] = 99
	fmt.Println(s[0], c[0])

	fmt.Println(slices.Insert([]int{1, 4}, 1, 2, 3))
	fmt.Println(slices.Delete([]int{1, 2, 3, 4}, 1, 3))
	fmt.Println(slices.Compact([]int{1, 1, 2, 2, 3}))
	fmt.Println(slices.Concat([]int{1}, []int{2, 3}))
	fmt.Println(slices.Repeat([]int{1, 2}, 3))
	fmt.Println(slices.Replace([]int{1, 2, 3}, 1, 2, 9, 9))

	r := []int{1, 2, 3}
	slices.Reverse(r)
	fmt.Println(r)

	words := []string{"Banana", "apple", "Cherry"}
	slices.SortFunc(words, func(a, b string) int {
		return strings.Compare(strings.ToLower(a), strings.ToLower(b))
	})
	fmt.Println(words)

	fmt.Println(slices.IndexFunc(s, func(v int) bool { return v > 5 }))
	fmt.Println(slices.DeleteFunc([]int{1, 2, 3, 4}, func(v int) bool { return v%2 == 0 }))
	fmt.Println(slices.MinFunc(words, strings.Compare), slices.MaxFunc(words, strings.Compare))

	var named ids = []int{3, 1, 2}
	slices.Sort(named)
	fmt.Println(named, slices.Contains(named, 2))

	// A long slice takes the merge path, and a stable sort keeps the order of
	// 2 equal values.
	long := make([]int, 40)
	for k := range long {
		long[k] = (40 - k) % 7
	}
	slices.Sort(long)
	fmt.Println(long[0], long[39], slices.IsSorted(long))

	fmt.Println(len(slices.Grow([]int{1}, 10)), cap(slices.Grow([]int{1}, 10)) >= 11)
	fmt.Println(cap(slices.Clip([]int{1, 2, 3}[:1])))
	fmt.Println(slices.Clone([]int(nil)) == nil)
}
