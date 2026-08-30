// The program proves that a make with no capacity reads its length expression
// 1 time. The expression below advances a counter, and a second read gives a
// different value, thus the make panics with 2 reads.
package main

import "fmt"

var calls int

func next() int {
	calls++
	return 10 - calls
}

func main() {
	for i := 0; i < 4; i++ {
		s := make([]byte, next())
		fmt.Println(len(s), cap(s))
	}
	fmt.Println("calls:", calls)
}
