package main

import (
	"cmp"
	"fmt"
)

// A type parameter with the constraint ~[]E holds the core type []E. Each
// operation of a slice applies to it.
func indexOf[S ~[]E, E comparable](s S, v E) int {
	for i := range s {
		if s[i] == v {
			return i
		}
	}
	return -1
}

func maxOf[S ~[]E, E cmp.Ordered](s S) E {
	out := s[0]
	for _, v := range s[1:] {
		if v > out {
			out = v
		}
	}
	return out
}

func grow[S ~[]E, E any](s S, n int) S {
	out := make(S, 0, len(s)+n)
	out = append(out, s...)
	return out
}

func clone[S ~[]E, E any](s S) S {
	if s == nil {
		return nil
	}
	return append(S(nil), s...)
}

// A type parameter with the constraint ~map[K]V holds the core type of a map.
func keyCount[M ~map[K]V, K comparable, V any](m M) int {
	n := 0
	for range m {
		n++
	}
	return n
}

type ids []int

type counts map[string]int

func main() {
	fmt.Println(indexOf([]string{"a", "b"}, "b"))
	fmt.Println(maxOf([]int{3, 9, 1}), maxOf([]string{"a", "z"}))

	var v ids = []int{5, 2, 8}
	fmt.Println(indexOf(v, 8), maxOf(v))

	g := grow(v, 10)
	fmt.Println(len(g), cap(g) >= 13)

	c := clone(v)
	c[0] = 99
	fmt.Println(v[0], c[0])
	fmt.Println(clone(ids(nil)) == nil)

	var m counts = map[string]int{"a": 1, "b": 2}
	fmt.Println(keyCount(m), keyCount(map[int]bool{1: true}))
}
