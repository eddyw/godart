// valuecopy proves the value semantics of a struct and of an array at each
// copy point that stage 4 knows.
package main

import "fmt"

type point struct{ X, Y int }

type holder struct{ p point }

func (h *holder) get() point { return h.p }

func main() {
	m := map[string]point{"a": {X: 1}}
	for k, v := range m {
		v.X = 99
		_ = k
	}
	fmt.Println("map value:", m["a"].X)

	mk := map[point]int{{X: 1}: 7}
	for k := range mk {
		k.X = 98
	}
	for k, n := range mk {
		fmt.Println("map key:", k.X, n)
	}

	s := []point{{X: 1}}
	for _, v := range s {
		v.X = 97
	}
	fmt.Println("slice value:", s[0].X)

	a := [2]point{{X: 1}, {X: 2}}
	for _, v := range a {
		v.X = 96
	}
	fmt.Println("array value:", a[0].X, a[1].X)

	h := &holder{p: point{X: 1}}
	var boxed any = h.get()
	h.p.X = 42
	fmt.Println("box of a call:", boxed, h.p.X)

	q := h.get()
	q.X = 5
	fmt.Println("call result:", h.p.X, q.X)

	b := a
	b[0].X = 11
	fmt.Println("array assign:", a[0].X, b[0].X)

	g := make([]point, 1, 1)
	g[0].X = 1
	grown := append(g, point{X: 2})
	grown[0].X = 99
	fmt.Println("append that grows:", g[0].X, grown[0].X)

	pin := &g[0]
	fmt.Println("the pointer of the old array:", pin.X)

	room := make([]point, 1, 4)
	room[0].X = 1
	shared := append(room, point{X: 2})
	shared[0].X = 88
	fmt.Println("append with room:", room[0].X, shared[0].X)

	src := []point{{X: 4}, {X: 5}}
	spread := append([]point{}, src...)
	spread[0].X = 77
	fmt.Println("append of a spread:", src[0].X, spread[0].X)

	sweep()
}
