// The program proves the rules of the package slices that hold a value which
// a hand written form gives in another way: the clear of the tail of a Delete,
// the bound of an Insert and of a Delete, the nil result of an empty Concat,
// and the overlap of the source and the destination.
package main

import (
	"fmt"
	"slices"
)

func try(name string, f func()) {
	defer func() {
		if e := recover(); e != nil {
			fmt.Printf("%-22s panic %v\n", name, e)
		}
	}()
	f()
}

func main() {
	s := []int{1, 2, 3, 4}
	out := slices.Delete(s, 0, 2)
	fmt.Println("delete", out, s)
	s2 := []int{1, 2, 3, 4, 5}
	fmt.Println("delete mid", slices.Delete(s2, 1, 3), s2)
	fmt.Printf("concat %#v %v\n", slices.Concat[[]int](), slices.Concat[[]int]() == nil)
	try("insert out of range", func() { slices.Insert([]int{1, 2}, 3, 9) })
	try("delete out of range", func() { slices.Delete([]int{1, 2}, 1, 3) })
	try("delete i>j", func() { slices.Delete([]int{1, 2}, 2, 1) })
	a := []int{1, 2, 3, 4, 5, 6}
	fmt.Println("insert overlap", slices.Insert(a[:3], 1, a[3:5]...))
	b := make([]int, 4, 10)
	copy(b, []int{9, 8, 7, 6})
	fmt.Println("replace", slices.Replace(b, 1, 3, b[2], b[3]))
	fmt.Println("grow", len(slices.Grow([]int{1}, 5)), cap(slices.Grow([]int{1}, 5)) >= 6)
	fmt.Println("clip", slices.Clip(make([]int, 2, 9)))
	fmt.Println("repeat", slices.Repeat([]int{1, 2}, 3))
	fmt.Println("compact", slices.Compact([]int{1, 1, 2, 2, 3}))
	fmt.Println("reverse", func() []int { r := []int{1, 2, 3}; slices.Reverse(r); return r }())
}
