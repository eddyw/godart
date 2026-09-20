package main

import "fmt"

type item struct {
	n int
}

func main() {
	var calls []int
	value := func(n int) int { calls = append(calls, n); return n }
	keyed := [8]int{7: value(7), 1: value(1), value(2)}
	fmt.Println("keyed", keyed, calls)
	structs := [3]item{2: {n: 9}, 0: {n: 1}}
	structs[0].n = 4
	fmt.Println("structs", structs[0].n, structs[1].n, structs[2].n)

	var a [3]int
	a[0] = 1
	a[2] = 3
	println(len(a), a[0], a[1], a[2])

	b := a
	b[0] = 100
	println(a[0], b[0])

	println(a == b, a == [3]int{1, 0, 3})

	c := [2]string{"x", "y"}
	println(c[1])

	s := a[1:3]
	println(len(s), s[0])
	s[0] = 55
	println(a[1])

	var items [2]item
	items[0].n = 7
	other := items
	other[0].n = 9
	println(items[0].n, other[0].n)

	for i, v := range c {
		println(i, v)
	}
}
