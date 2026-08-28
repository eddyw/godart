package main

import "fmt"

type point struct{ x, y int }

type node struct {
	next  *node
	count *int
}

type tally int

func bump(p *int) { *p = *p + 1 }

func swap(a, b *string) { *a, *b = *b, *a }

func sum(vals ...*int) int {
	out := 0
	for _, v := range vals {
		out += *v
	}
	return out
}

// A parameter whose address the body takes needs a box.
func viaParam(n int) int {
	p := &n
	*p = *p * 2
	return n
}

// A closure that writes a boxed variable of its owner must see the write.
func viaClosure() int {
	n := 1
	p := &n
	f := func() { *p = 9 }
	f()
	return n
}

// A named result whose address the body takes needs a box.
func viaResult() (out int) {
	p := &out
	*p = 7
	return
}

// A receiver that is not a struct needs a box.
func (t tally) doubled() int {
	p := &t
	*p = *p * 2
	return int(*p)
}

func main() {
	n := 5
	bump(&n)
	fmt.Println(n)

	p := &n
	*p = 42
	fmt.Println(n, *p)

	q := new(int)
	*q = 7
	fmt.Println(*q)
	fmt.Println(*new(string) == "", *new(bool))

	x, y := "a", "b"
	swap(&x, &y)
	fmt.Println(x, y)

	var nilp *int
	fmt.Println(nilp == nil, p != nil)

	s := []int{1, 2}
	sp := &s
	*sp = append(*sp, 3)
	fmt.Println(s, len(*sp))

	*p += 8
	fmt.Println(n)

	f := 1.5
	fp := &f
	*fp *= 2
	fmt.Println(f)

	fmt.Println(viaParam(5), viaClosure(), viaResult())
	fmt.Println(tally(4).doubled())

	a, b := 1, 2
	fmt.Println(sum(&a, &b))

	// A pointer inside a struct field works, because the field holds the box.
	c := 3
	nd := node{count: &c}
	*nd.count = 4
	fmt.Println(c, *nd.count)

	// A slice of pointers and a map of pointers.
	ps := []*int{&a, &b}
	*ps[0] = 10
	fmt.Println(a, *ps[0])
	mp := map[string]*int{"a": &a}
	*mp["a"] = 11
	fmt.Println(a)

	// A pointer to a pointer.
	pp := &ps
	fmt.Println(len(*pp))

	// The identity of 2 pointers.
	fmt.Println(&a == &a, &a == &b)

	// A boxed variable inside a loop keeps its own box for each round.
	total := 0
	for i := 0; i < 3; i++ {
		v := i
		vp := &v
		*vp += 10
		total += v
	}
	fmt.Println(total)

	// A pointer to a struct still holds the object itself.
	pt := point{1, 2}
	ptp := &pt
	ptp.x = 9
	fmt.Println(pt.x)
}
