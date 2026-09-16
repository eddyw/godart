package main

import (
	"fmt"
	"unique"
)

type Point struct{ X, Y int }
type Named int

func same[T comparable](a, b T) {
	x, y := unique.Make(a), unique.Make(b)
	fmt.Println(x == y, x.Value() == a, y.Value() == b)
	fmt.Println(map[unique.Handle[T]]int{x: 1}[y])
}
func main() {
	same(1, 1)
	same(1, 2)
	same(Named(1), Named(1))
	same("abc", "abc")
	same(Point{1, 2}, Point{1, 2})
	same(Point{1, 2}, Point{1, 3})
	same([2]int{1, 2}, [2]int{1, 2})
	same([2][2]int{{1, 2}}, [2][2]int{{1, 2}})
	p, q := &Point{1, 2}, &Point{1, 2}
	same(p, p)
	same(p, q)
	same([1]*Point{p}, [1]*Point{q})
	fmt.Println(pointerEqual(p, p), pointerEqual(p, q))
	same[any](Named(1), 1)
	var nilPointer *int
	same(nilPointer, nilPointer)
	var zero unique.Handle[int]
	fmt.Println(zero == unique.Handle[int]{}, zero == unique.Make(0))
}

func pointerEqual[T any](a, b *T) bool { return a == b }
