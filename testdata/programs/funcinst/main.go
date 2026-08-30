package main

import "maps"

func equal[V comparable](a, b V) bool { return a == b }

func first[T any](s []T) T { return s[0] }

func swap[K comparable, V any](k K, v V) (V, K) { return v, k }

func sum[T ~int | ~float64](vs ...T) T {
	var out T
	for _, v := range vs {
		out += v
	}
	return out
}

func recvOf[T any](c chan T) T { return <-c }

func apply[T any](f func(T, T) bool, a, b T) bool { return f(a, b) }

func main() {
	f := equal[int]
	println(f(1, 1), f(1, 2))

	// The type checker infers the type argument from the target type.
	var g func(string, string) bool = equal
	println(g("a", "a"))

	println(apply(equal[string], "a", "b"))
	println(first[int]([]int{7, 8}))

	var s func(int, string) (string, int) = swap
	text, n := s(3, "k")
	println(text, n)

	var vs func(...float64) float64 = sum
	println(vs(1.5, 2.5))
	rest := []int{4, 5, 6}
	var is func(...int) int = sum
	println(is(rest...))

	// A generic function that blocks reaches a value.
	var h func(chan int) int = recvOf[int]
	c := make(chan int)
	go func() { c <- 42 }()
	println(h(c))

	table := []func(int, int) bool{equal[int], func(a, b int) bool { return a < b }}
	println(table[0](2, 2), table[1](1, 2))

	m1 := map[string]int{"a": 1}
	m2 := map[string]int{"a": 1}
	println(maps.EqualFunc(m1, m2, equal[int]))
}
