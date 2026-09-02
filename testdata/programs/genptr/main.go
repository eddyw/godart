package main

import "fmt"

type pt struct{ X, Y int }

// ptr gives a pointer to a copy of the value. The body holds no type argument,
// thus it writes a cell. The Dart form of a pointer follows the type argument,
// and the call site reads that cell out.
func ptr[T any](v T) *T { return &v }

// pick gives the first pointer that is not nil.
func pick[T any](a, b *T) *T {
	if a != nil {
		return a
	}
	return b
}

func main() {
	p := ptr(pt{X: 1, Y: 2})
	fmt.Println(p.X, p.Y)
	p.X = 7
	fmt.Println(*p)

	n := ptr(41)
	*n++
	fmt.Println(*n)

	s := ptr("go")
	fmt.Println(*s)

	a := ptr([2]int{3, 4})
	a[1] = 9
	fmt.Println(*a)

	var none *pt
	fmt.Println(pick(none, p).Y)
	fmt.Println(pick[pt](nil, nil) == nil)

	// A pointer that a generic function gives must reach the same value.
	q := pick(p, none)
	q.Y = 5
	fmt.Println(p.Y)
}
