package main

import "fmt"

type Box[T any] struct{ V T }

func (b Box[T]) Get() T { return b.V }

func (b Box[T]) String() string { return fmt.Sprint("box of ", b.V) }

func (b *Box[T]) Set(v T) { b.V = v }

type stringer interface{ String() string }

func main() {
	var s stringer = Box[int]{V: 3}
	fmt.Println(s.String())

	var s2 stringer = Box[string]{V: "a"}
	fmt.Println(s2.String())

	var a any = Box[string]{V: "x"}
	if b, ok := a.(Box[string]); ok {
		fmt.Println("assert:", b.Get())
	}
	if _, ok := a.(Box[int]); ok {
		fmt.Println("the wrong instance matched")
	}

	fmt.Printf("%T %v\n", a, a)

	switch v := a.(type) {
	case Box[int]:
		fmt.Println("int box", v.V)
	case Box[string]:
		fmt.Println("string box", v.V)
	}

	p := &Box[int]{V: 7}
	var ps any = p
	if q, ok := ps.(*Box[int]); ok {
		q.Set(8)
		fmt.Println("pointer:", p.V)
	}
}
