package main

import (
	"fmt"
	"math"
)

type addable interface {
	~int8 | ~uint64 | ~float32 | ~complex64 | ~string
}
type text string
type small int8

func add[T addable](a, b T) T                           { return a + b }
func addTo[T addable](values []T, next func() int, a T) { values[next()] += a }
func sum[T addable](values ...T) T {
	var result T
	for _, value := range values {
		result += value
	}
	return result
}
func main() {
	fmt.Println(add[int8](127, 1), add[small](127, 2), add[uint64](math.MaxUint64, 2))
	fmt.Println(add[float32](16777216, 1), add[complex64](16777216+2i, 1+3i))
	fmt.Printf("%q %q\n", add("a\x00\xff", "世界"), add(text("hello"), text(" there")))
	fmt.Println(sum[int8](120, 8, 1), sum[float32](16777216, 1, -16777216))
	fmt.Println(sum[complex64](16777216+2i, 1+3i, -16777216))
	fmt.Printf("%q %q\n", sum[text]("a", "\xff", "b"), sum[string]())
	n := 0
	next := func() int { n++; return 0 }
	strings := []text{"first"}
	numbers := []small{127}
	addTo(strings, next, text(" second"))
	addTo(numbers, next, small(2))
	fmt.Println(strings[0], numbers[0], n)
}
