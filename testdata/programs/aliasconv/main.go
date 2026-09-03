// Command aliasconv proves a conversion to a type that is the same type.
// Go accepts such a conversion, and it gives a copy of the value. An alias of
// a struct type and an alias of an array type each make 1.
package main

import "fmt"

type point struct{ X, Y int }

// spot is an alias of point. Each conversion below reads the same type 2
// times, thus the conversion changes no type and only copies.
type spot = point

// grid is an alias of an array type.
type grid = [2]point

// tag is an alias of a named struct type of another shape.
type tag = struct{ Name string }

func main() {
	a := point{X: 1, Y: 2}
	b := spot(a)
	b.X = 33
	fmt.Println("the alias of a struct:", a.X, b.X, b.Y)

	c := point(a)
	c.Y = 44
	fmt.Println("the name of the type itself:", a.Y, c.Y)

	rows := [2]point{{X: 1}, {X: 2}}
	copied := grid(rows)
	copied[0].X = 55
	fmt.Println("the alias of an array:", rows[0].X, copied[0].X)

	plain := [2]point(rows)
	plain[1].X = 66
	fmt.Println("the array type itself:", rows[1].X, plain[1].X)

	t := tag{Name: "one"}
	u := tag(t)
	u.Name = "two"
	fmt.Println("the alias of an unnamed struct:", t.Name, u.Name)

	// A conversion inside a call takes the copy too, thus the write of the
	// callee reaches no value of the caller.
	fmt.Println("through a call:", grow(spot(a)), a.X)
}

// grow writes the copy that the caller gave.
func grow(p spot) int {
	p.X += 100
	return p.X
}
