// The program proves 3 rules that an unnamed struct type and a parallel
// assignment hold. 2 shapes that differ in a field name are 2 types. A field
// of a pointer reads the pointer before the first write. A return of a
// function with named results reads each operand before the first write.
package main

import (
	"fmt"
	"reflect"
)

type Int = int

// shapes proves that the name of a field and the meaning of a local alias each
// belong to the identity of a shape.
func shapes() {
	var x, y any = struct{ int }{}, struct{ Int }{}
	fmt.Println("embedded name", x == y)
	{
		type C = int32
		x = struct{ C }{}
	}
	{
		type C = uint32
		y = struct{ C }{}
	}
	fmt.Println("local alias", x == y)
	var a, b any = struct{ n int }{1}, struct{ n int }{1}
	fmt.Println("equal shapes", a == b)
	fmt.Printf("%T %T %T\n", struct{ int }{}, struct{ Int }{}, struct{ n int }{})
}

// marks proves that an unexported field of an unnamed struct type carries the
// read only mark of reflect.
func marks() {
	var T struct {
		int
		Name string
		age  int
	}
	v := reflect.ValueOf(&T).Elem()
	for i := 0; i < v.NumField(); i++ {
		f := v.Type().Field(i)
		fmt.Printf("field %-6s exported=%-6v path=%-10q canset=%v\n",
			f.Name, f.IsExported(), f.PkgPath, v.Field(i).CanSet())
	}
}

type node struct{ i int }

type deep struct{ x struct{ y int } }

// order proves that a parallel assignment reads the base of each field of a
// pointer before the first write.
func order() {
	var a node
	p := &a
	p, p.i = new(node), 4
	fmt.Println("field of a pointer", a.i, p.i)
	var d deep
	q := &d
	q, q.x.y = new(deep), 7
	fmt.Println("nested field", d.x.y, q.x.y)
	m := map[int]int{}
	var np *int
	func() {
		defer func() { recover() }()
		m[2], *np = 42, 2
	}()
	fmt.Println("map before the panic", len(m), m[2])
}

// results proves that a return reads each operand before it writes a result
// variable.
func results() {
	fmt.Println(named())
	fmt.Println(blank())
}

func named() (a bool, b int) {
	defer func() { recover() }()
	var p *int
	return true, *p
}

func blank() (_ bool, _ int) {
	defer func() { recover() }()
	var p *int
	return true, *p
}

func main() {
	shapes()
	marks()
	order()
	results()
}
