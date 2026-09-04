// The program proves that a pointer conversion between 2 defined struct types
// gives a pointer to the same value.
package main

import "fmt"

type inner struct {
	a int
	b string
}

type outer inner

func (o *outer) label() string {
	return fmt.Sprintf("outer(%d,%s)", o.a, o.b)
}

func (i *inner) label() string {
	return fmt.Sprintf("inner(%d,%s)", i.a, i.b)
}

func main() {
	p := &inner{a: 1, b: "one"}
	q := (*outer)(p)

	fmt.Println(p.label(), q.label())

	q.a = 2
	fmt.Println("write through q reaches p", p.a)

	p.b = "two"
	fmt.Println("write through p reaches q", q.b)

	back := (*inner)(q)
	fmt.Println("round trip", back == p)

	other := (*outer)(p)
	fmt.Println("2 conversions of 1 pointer", q == other)

	var face any = p
	_, isInner := face.(*inner)
	_, isOuter := face.(*outer)
	fmt.Println("identity of p", isInner, isOuter)

	face = q
	_, isInner = face.(*inner)
	_, isOuter = face.(*outer)
	fmt.Println("identity of q", isInner, isOuter)

	fmt.Printf("types %T %T\n", p, q)

	type namer interface{ label() string }
	fmt.Println(face.(namer).label())

	value := outer(*p)
	value.a = 3
	fmt.Println("a value conversion copies", p.a, value.a)
}
