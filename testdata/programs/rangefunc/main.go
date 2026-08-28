package main

import (
	"fmt"
	"iter"
)

func upTo(n int) iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := 0; i < n; i++ {
			if !yield(i) {
				return
			}
		}
	}
}

func pairs(m []string) iter.Seq2[int, string] {
	return func(yield func(int, string) bool) {
		for i, v := range m {
			if !yield(i, v) {
				return
			}
		}
	}
}

func main() {
	for v := range upTo(4) {
		fmt.Print(v, " ")
	}
	fmt.Println()

	// A break stops the sequence.
	for v := range upTo(10) {
		if v == 3 {
			break
		}
		fmt.Print(v, " ")
	}
	fmt.Println()

	// A continue skips 1 value.
	for v := range upTo(6) {
		if v%2 == 0 {
			continue
		}
		fmt.Print(v, " ")
	}
	fmt.Println()

	for i, v := range pairs([]string{"a", "b", "c"}) {
		fmt.Print(i, v, " ")
	}
	fmt.Println()

	// A nested loop keeps its own break and its own continue.
	for v := range upTo(3) {
		for j := 0; j < 3; j++ {
			if j == 1 {
				break
			}
			fmt.Print(v, j, " ")
		}
	}
	fmt.Println()

	// A switch inside the body catches its own break.
	for v := range upTo(3) {
		switch v {
		case 1:
			break
		default:
			fmt.Print(v, " ")
		}
	}
	fmt.Println()

	// A range loop with no variable counts the values.
	n := 0
	for range upTo(5) {
		n++
	}
	fmt.Println(n)

	// A nested sequence works.
	for a := range upTo(2) {
		for b := range upTo(2) {
			fmt.Print(a, b, " ")
		}
	}
	fmt.Println()

	returns()
	fmt.Println(breakOuter())
	fmt.Println(continueOuter())
	fmt.Println(deep())
}

// breakOuter leaves a loop outside the body of the sequence. The body is a Dart
// closure, thus the branch writes a flag and gives false to the sequence.
func breakOuter() []int {
	var out []int
outer:
	for row := 0; row < 3; row++ {
		for v := range upTo(3) {
			if row == 1 && v == 1 {
				break outer
			}
			out = append(out, row*10+v)
		}
	}
	return out
}

func continueOuter() []int {
	var out []int
outer:
	for row := 0; row < 3; row++ {
		for v := range upTo(3) {
			if v == 1 {
				continue outer
			}
			out = append(out, row*10+v)
		}
		out = append(out, 99)
	}
	return out
}

// deep leaves 2 bodies. The flag travels out 1 level for each sequence.
func deep() []int {
	var out []int
outer:
	for row := 0; row < 2; row++ {
		for a := range upTo(2) {
			for b := range upTo(2) {
				if a == 1 && b == 1 {
					break outer
				}
				out = append(out, row*100+a*10+b)
			}
		}
	}
	return out
}
