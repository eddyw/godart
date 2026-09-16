package main

import (
	"fmt"
	"slices"
)

type item struct{ A, B int }

func swap[T any](a, b *T) { *a, *b = *b, *a }

func copyValue[T any](a T, replacement T) T {
	b := a
	a = replacement
	return b
}

func main() {
	a, b := item{1, 2}, item{3, 4}
	p := &a.A
	swap(&a, &b)
	fmt.Println(a, b, *p, copyValue(a, b))
	x := []item{{3, 30}, {1, 10}, {2, 20}}
	slices.SortFunc(x, func(a, b item) int { return a.A - b.A })
	fmt.Println(x)
	u, v := [2]int{1, 2}, [2]int{3, 4}
	swap(&u, &v)
	fmt.Println(u, v, copyValue(u, v))
}
