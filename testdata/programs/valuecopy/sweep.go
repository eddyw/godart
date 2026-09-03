package main

import "fmt"

type box struct {
	P point
	A [2]point
}

type reader interface{ get() int }

func (p point) get() int { return p.X }

func take(p point) { p.X = 100 }

func vary(ps ...point) { ps[0].X = 101 }

var global = point{X: 1}

func two() (point, point) { return global, global }

// sweep reads each place where Go copies a struct or an array value.
func sweep() {
	a := point{X: 1}
	b := a
	b.X = 2
	fmt.Println("assign:", a.X, b.X)

	take(a)
	fmt.Println("argument:", a.X)

	vary(a)
	fmt.Println("variadic:", a.X)

	m := map[string]point{"k": {X: 1}}
	v := m["k"]
	v.X = 3
	fmt.Println("map read:", m["k"].X, v.X)

	ch := make(chan point, 1)
	ch <- a
	a.X = 4
	got := <-ch
	fmt.Println("channel:", got.X, a.X)

	var i reader = a
	a.X = 5
	fmt.Println("interface:", i.get(), a.X)

	c := a
	f := func() { c.X = 6 }
	f()
	fmt.Println("capture:", a.X, c.X)

	x, y := two()
	x.X = 7
	fmt.Println("2 results:", global.X, x.X, y.X)

	bx := box{P: point{X: 1}, A: [2]point{{X: 1}, {X: 2}}}
	cp := bx
	cp.P.X = 8
	cp.A[0].X = 9
	fmt.Println("nested:", bx.P.X, bx.A[0].X, cp.P.X, cp.A[0].X)

	arr := [2]point{{X: 1}, {X: 2}}
	m2 := map[string][2]point{"a": arr}
	arr[0].X = 10
	fmt.Println("array in a map:", m2["a"][0].X, arr[0].X)

	s := []point{{X: 1}, {X: 2}}
	d := make([]point, 2)
	copy(d, s)
	d[0].X = 11
	fmt.Println("copy:", s[0].X, d[0].X)

	copy(s[1:], s[:1])
	s[1].X = 12
	fmt.Println("overlap:", s[0].X, s[1].X)

	st := struct{ P point }{P: point{X: 1}}
	st2 := st
	st2.P.X = 13
	fmt.Println("unnamed struct:", st.P.X, st2.P.X)

	l := global
	l.X = 14
	fmt.Println("global:", global.X, l.X)

	ptrs := []*point{{X: 1}}
	deref := *ptrs[0]
	deref.X = 15
	fmt.Println("deref:", ptrs[0].X, deref.X)

	list := []point{{X: 1}}
	for i := range list {
		e := list[i]
		e.X = 16
	}
	fmt.Println("index read:", list[0].X)

	mm := map[point]point{{X: 1}: {Y: 1}}
	for k, w := range mm {
		k.X = 17
		w.Y = 18
	}
	for k, w := range mm {
		fmt.Println("map pair:", k.X, w.Y)
	}
}
