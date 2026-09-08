// The witness of the bound reports of slices and of the clear of a tail.
package main

import (
	"fmt"
	"slices"
)

func try(name string, f func()) {
	defer func() {
		if e := recover(); e != nil {
			fmt.Printf("%s: panic %v\n", name, e)
			return
		}
		fmt.Printf("%s: no panic\n", name)
	}()
	f()
}

func main() {
	try("insert", func() { _ = slices.Insert([]int{1, 2}, 3, 9) })
	try("delete", func() { _ = slices.Delete([]int{1, 2}, 1, 3) })

	s := []int{1, 2, 3, 4}
	out := slices.Delete(s, 0, 2)
	fmt.Println("delete tail", out, s)

	var empty []int
	c := slices.Concat[[]int]()
	fmt.Println("concat nil", c == nil, len(c), empty == nil)

	r := slices.Repeat([]int{1, 2}, 2)
	fmt.Println("repeat", r, cap(r))

	t := []int{5, 1, 4}
	slices.Sort(t)
	fmt.Println("sort", t)
	fmt.Println("compact", slices.Compact([]int{1, 1, 2, 2, 3}))
	fmt.Println("clip", cap(slices.Clip(make([]int, 2, 8))))
}
