package main

import "fmt"

type counter struct {
	n    int
	name string
	next *counter
}

func bump(p *int) { *p += 10 }

func setName(p *string, text string) { *p = text }

func main() {
	c := counter{n: 1, name: "a"}
	p := &c.n
	*p = 5
	fmt.Println(c.n, *p)
	bump(p)
	fmt.Println(c.n, *p)
	*p++
	fmt.Println(c.n)

	q := &c.name
	setName(q, "b")
	fmt.Println(c.name, *q)

	other := counter{}
	fmt.Println(p == &c.n, q == &c.name, p == &other.n)

	s := []int{1, 2, 3}
	e := &s[1]
	*e = 20
	fmt.Println(s, *e, e == &s[1], e == &s[2])
	bump(e)
	fmt.Println(s)

	var a [3]string
	ap := &a[2]
	*ap = "last"
	fmt.Println(a, *ap, ap == &a[2])

	t := s[1:]
	fmt.Println(&t[0] == &s[1])

	d := &c
	dp := &d.n
	*dp = 100
	fmt.Println(c.n, dp == p)

	list := []counter{{n: 1}, {n: 2}}
	lp := &list[0].n
	*lp = 9
	fmt.Println(list[0].n)

	var pp *int
	fmt.Println(pp == nil)
	pp = &c.n
	fmt.Println(*pp)
}
