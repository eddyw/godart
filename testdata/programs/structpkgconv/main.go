// The program proves a defined type over a struct type of another package. Go
// gives the 2 names 1 struct object, thus a conversion between the 2 names 1
// value and a pointer conversion needs no operation. The emitter gives the 2
// names 1 Dart class, and that class stands in the library of the package that
// declares the object.
package main

import (
	"fmt"
	"image"
	"time"
)

// pt reads the struct object of image.Point. The 2 fields of that object are
// exported, thus a literal of the type stands here.
type pt image.Point

// shift moves the point. The receiver is a pointer, thus the write reaches the
// value of the caller.
func (p *pt) shift(dx, dy int) {
	p.X += dx
	p.Y += dy
}

// stamp reads the struct object of time.Time. Each field of that object
// belongs to the package time, thus no code here names one.
type stamp time.Time

// set writes the time into the receiver. The conversion carries the whole
// value, and no field of it reaches this package.
func (s *stamp) set(t time.Time) { *s = stamp(t) }

type mover interface{ shift(int, int) }

func main() {
	p := pt{X: 1, Y: 2}
	q := image.Point(p)
	fmt.Println(q.X, q.Y, q.String())

	back := pt(q)
	back.shift(3, 4)
	fmt.Println(back.X, back.Y, p.X, p.Y)

	var m mover = &back
	m.shift(1, 1)
	fmt.Println(back.X, back.Y)
	fmt.Printf("%T %T %v\n", p, q, p)

	// A conversion stands at a copy point. A write through the source thus
	// reaches no field of the result.
	r := &pt{X: 5, Y: 6}
	s := image.Point(*r)
	r.shift(1, 1)
	fmt.Println(s.X, s.Y, r.X, r.Y)

	// A pointer conversion names 1 value, thus the write reaches r.
	ip := (*image.Point)(r)
	ip.X = 9
	fmt.Println(r.X, ip == (*image.Point)(r))

	var st stamp
	st.set(time.Date(2026, 9, 6, 1, 2, 3, 0, time.UTC))
	fmt.Println(time.Time(st).Format(time.RFC3339))
	fmt.Printf("%T\n", st)

	// The equality of the class reads each field of the struct object.
	fmt.Println(pt{X: 1, Y: 2} == p, image.Point(p) == q)

	var a any = p
	_, isPoint := a.(image.Point)
	other, isPt := a.(pt)
	fmt.Println(isPoint, isPt, other.X)
}
