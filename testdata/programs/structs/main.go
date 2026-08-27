package main

type point struct {
	x, y int
}

type line struct {
	a point
	b point
}

func (p point) sum() int {
	return p.x + p.y
}

func (p point) move(dx int) point {
	p.x += dx
	return p
}

func (p *point) scale(k int) {
	p.x *= k
	p.y *= k
}

func main() {
	a := point{1, 2}
	b := a
	b.x = 100
	println(a.x, b.x, a.sum())

	c := a.move(10)
	println(a.x, c.x)

	p := &a
	p.scale(3)
	println(a.x, a.y, p.sum())

	l := line{a: point{1, 2}, b: point{x: 3}}
	l2 := l
	l2.a.x = 9
	println(l.a.x, l2.a.x, l.b.x, l.b.y)

	q := &l.a
	q.y = 42
	println(l.a.y)

	println(a == point{3, 6})
	var zero point
	println(zero.x, zero.y)

	n := new(point)
	n.x = 7
	println(n.x, n.y)
}
