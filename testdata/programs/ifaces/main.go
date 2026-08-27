package main

type shape interface {
	area() int
	name() string
}

type namer interface {
	name() string
}

type rect struct {
	w, h int
}

func (r rect) area() int    { return r.w * r.h }
func (r rect) name() string { return "rect" }

type square struct {
	side int
}

func (s *square) area() int    { return s.side * s.side }
func (s *square) name() string { return "square" }

type counter int

func (c counter) name() string { return "counter" }

type base struct {
	id int
}

func (b base) name() string { return "base" }

type derived struct {
	base
	extra int
}

func describe(x any) string {
	switch v := x.(type) {
	case int:
		if v > 10 {
			return "big int"
		}
		return "int"
	case string:
		return "string " + v
	case rect:
		return "rect " + v.name()
	case nil:
		return "nil"
	case namer:
		return "namer " + v.name()
	}
	return "other"
}

func main() {
	var s shape = rect{3, 4}
	println(s.area(), s.name())

	s = &square{5}
	println(s.area(), s.name())

	var n namer = s
	println(n.name())

	shapes := []shape{rect{1, 2}, &square{3}}
	total := 0
	for _, sh := range shapes {
		total += sh.area()
	}
	println(total)

	println(describe(5), describe(50))
	println(describe("x"))
	println(describe(rect{1, 1}))
	println(describe(nil))
	println(describe(counter(7)))
	println(describe(1.5))

	var any1 any = 42
	v, ok := any1.(int)
	println(v, ok)
	_, ok2 := any1.(string)
	println(ok2)
	println(any1.(int) + 1)

	d := derived{base{9}, 1}
	println(d.name(), d.id)
	var nm namer = d
	println(nm.name())

	var empty shape
	println(empty == nil)

	var a any = rect{1, 2}
	var b any = rect{1, 2}
	println(a == b)
}
