// The program proves the value and the expression of a generic method. The
// method n carries its own type parameters, and the value must bind their
// dictionaries. The method wait blocks, thus the value of it must keep the
// block model.
package main

import "fmt"

type pair[A, B any] struct {
	a A
	b B
}

func (p pair[A, B]) tag() string {
	return fmt.Sprintf("%T/%T", p.a, p.b)
}

func (p pair[A, B]) both[C, D any]() string {
	var c C
	var d D
	return fmt.Sprintf("%T/%T/%T/%T", p.a, p.b, c, d)
}

func (p *pair[A, B]) wait[C any](ch chan C) C {
	return <-ch
}

func main() {
	p := pair[int8, int16]{}

	// The value of a generic method with a value receiver.
	v1 := p.both[int32, int64]
	fmt.Println(v1())
	v2 := p.both[string, bool]
	fmt.Println(v2())

	// The expression of a generic method.
	e1 := pair[int8, int16].both[int32, string]
	fmt.Println(e1(p))

	// The value of a generic method inside a generic function.
	fmt.Println(make3[uint8, uint16, uint32]()())

	// The value of a generic method with a pointer receiver, and a block
	// operation inside the method.
	q := &pair[int, int]{}
	ch := make(chan string, 1)
	w := q.wait[string]
	go func() { ch <- "sent" }()
	fmt.Println(w(ch))
}

func make3[A, B, C any]() func() string {
	return pair[A, B]{}.both[C, C]
}
