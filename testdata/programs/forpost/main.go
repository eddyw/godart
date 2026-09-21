package main

import "fmt"

func main() { exercise(); fmt.Println("ok") }

func exercise() {
	m := map[int]int{}
	for i := 0; i < 3; m[i] = i * 10 {
		i++
		if i == 2 {
			continue
		}
	}
	if len(m) != 3 || m[1] != 10 || m[2] != 20 || m[3] != 30 {
		panic(m)
	}
	s := make([]int, 4)
	for i := 0; i < 3; s[i] = i {
		i++
		continue
	}
	if s[1] != 1 || s[2] != 2 || s[3] != 3 {
		panic(s)
	}
	a := [4]int{}
	for i := 0; i < 3; a[i] = i {
		i++
	}
	if a != [4]int{0, 1, 2, 3} {
		panic(a)
	}
	b := make([]byte, 4)
	for i := 0; i < 3; b[i] = byte(i) {
		i++
	}
	if string(b) != "\x00\x01\x02\x03" {
		panic(b)
	}
	p := &[4]int{}
	for i := 0; i < 3; p[i] = i {
		i++
	}
	if *p != a {
		panic(p)
	}
	for s[0] = 0; s[0] < 2; s[0] = s[0] + 1 {
	}
	if s[0] != 2 {
		panic(s)
	}
	calls := 0
	index := func() int { calls++; return calls }
	value := func() int { return calls * 2 }
	for i := 0; i < 2; m[index()] = value() {
		i++
	}
	if calls != 2 || m[1] != 2 || m[2] != 4 {
		panic(m)
	}
	for i := 0; i < 1; m[index()] = 0 {
		break
	}
	if calls != 2 {
		panic(calls)
	}
outer:
	for i := 0; i < 2; s[i] = i {
		i++
		for j := 0; j < 2; j++ {
			continue outer
		}
	}
	if s[1] != 1 || s[2] != 2 {
		panic(s)
	}
}
