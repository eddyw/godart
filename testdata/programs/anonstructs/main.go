package main

import "fmt"

type point struct{ x, y int }

func main() {
	v := struct {
		name string
		n    int
	}{name: "a", n: 1}
	w := v
	w.n = 2
	println(v.n, w.n)

	rows := []struct {
		name string
		in   int
		want int
	}{
		{name: "one", in: 1, want: 2},
		{name: "two", in: 2, want: 4},
	}
	for _, r := range rows {
		println(r.name, r.in*2 == r.want)
	}

	var zero struct {
		s string
		p point
		a [2]int
	}
	println(zero.s == "", zero.p.x, zero.a[1])

	arr := [2]struct{ p point }{{p: point{1, 2}}, {p: point{3, 4}}}
	b := arr
	b[0].p.x = 9
	println(arr[0].p.x, b[0].p.x)

	m := map[struct{ a int }]int{{a: 1}: 10}
	println(m[struct{ a int }{a: 1}])

	var any1 any = v
	fmt.Println(any1)
	if s, ok := any1.(struct {
		name string
		n    int
	}); ok {
		println("assert", s.name)
	}

	println(v == struct {
		name string
		n    int
	}{name: "a", n: 1})

	q := &rows[0]
	q.want = 99
	println(rows[0].want)
}
