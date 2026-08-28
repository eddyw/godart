package main

// This file holds each form of a return statement inside the body of a range
// loop over a function.

import "fmt"

// A return inside the body leaves the function.
func find(n, want int) int {
	for v := range upTo(n) {
		if v == want {
			return v
		}
	}
	return -1
}

// A return with 2 values.
func findPair(n, want int) (int, bool) {
	for v := range upTo(n) {
		if v == want {
			return v * 10, true
		}
	}
	return 0, false
}

// A bare return with a named result.
func findNamed(n, want int) (out int) {
	for v := range upTo(n) {
		if v == want {
			out = v * 100
			return
		}
	}
	return -1
}

// A return inside a nested sequence.
func findNested(n int) (int, int) {
	for a := range upTo(n) {
		for b := range upTo(n) {
			if a*b == 6 {
				return a, b
			}
		}
	}
	return -1, -1
}

// A return inside a yield body that a plain loop holds.
func findInLoop(n int) int {
	for i := 0; i < n; i++ {
		for v := range upTo(n) {
			if i+v == 4 {
				return i*10 + v
			}
		}
	}
	return -1
}

// A body with a return and a break.
func mixed(n int) int {
	total := 0
	for v := range upTo(n) {
		if v == 3 {
			break
		}
		if v == 9 {
			return -99
		}
		total += v
	}
	return total
}

func returns() {
	fmt.Println(find(5, 2), find(5, 9))
	fmt.Println(findPair(5, 3))
	fmt.Println(findPair(5, 9))
	fmt.Println(findNamed(5, 2), findNamed(5, 9))
	fmt.Println(findNested(4))
	fmt.Println(findInLoop(5))
	fmt.Println(mixed(6))
}
