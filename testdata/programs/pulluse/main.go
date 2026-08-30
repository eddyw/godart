package main

import (
	"fmt"
	"iter"
	"maps"
	"slices"
)

func count(n int) iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := 0; i < n; i++ {
			if !yield(i) {
				return
			}
		}
	}
}

func pairs() iter.Seq2[string, int] {
	return func(yield func(string, int) bool) {
		for i, name := range []string{"a", "b", "c"} {
			if !yield(name, i) {
				return
			}
		}
	}
}

// merge reads 2 sequences at the same time, and it gives the pairs.
func merge(a, b iter.Seq[int]) []string {
	nextA, stopA := iter.Pull(a)
	defer stopA()
	nextB, stopB := iter.Pull(b)
	defer stopB()
	var out []string
	for {
		x, okX := nextA()
		y, okY := nextB()
		if !okX && !okY {
			return out
		}
		out = append(out, fmt.Sprintf("%d/%v %d/%v", x, okX, y, okY))
	}
}

func main() {
	next, stop := iter.Pull(count(4))
	for {
		v, ok := next()
		if !ok {
			break
		}
		fmt.Println("value", v)
	}
	stop()
	v, ok := next()
	fmt.Println("after the end", v, ok)

	// A stop before the end of the sequence ends it.
	early, stopEarly := iter.Pull(count(10))
	first, _ := early()
	second, _ := early()
	stopEarly()
	third, okThird := early()
	fmt.Println("early", first, second, third, okThird)

	next2, stop2 := iter.Pull2(pairs())
	defer stop2()
	for {
		k, n, ok := next2()
		if !ok {
			break
		}
		fmt.Println("pair", k, n)
	}

	for _, line := range merge(count(2), count(4)) {
		fmt.Println("merge", line)
	}

	// A sequence of the standard library reaches Pull.
	m := map[string]int{"x": 1, "y": 2, "z": 3}
	keys, stopKeys := iter.Pull(maps.Keys(m))
	var got []string
	for {
		k, ok := keys()
		if !ok {
			break
		}
		got = append(got, k)
	}
	stopKeys()
	slices.Sort(got)
	fmt.Println("keys", got)

	// A stop with no call of next ends the sequence.
	_, stopNone := iter.Pull(count(3))
	stopNone()
	fmt.Println("done")
}
