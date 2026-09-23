package main

import (
	"fmt"

	"example.com/genericstructidentity/foreign"
)

func create[T any](v T) any { return struct{ X T }{v} }
func read[T any](v any)     { x, ok := v.(struct{ X T }); fmt.Println(x.X, ok) }
func same[T any](v T) any {
	return struct {
		X T
		Y T
	}{v, v}
}
func pair[A, B any](v any) {
	x, ok := v.(struct {
		X A
		Y B
	})
	fmt.Println(x.X, x.Y, ok)
}
func nested[T any](v any) {
	x, ok := v.(struct {
		X []T
		Y struct{ Z T }
	})
	fmt.Println(x.X[0], x.Y.Z, ok)
}
func main() {
	pair[int, int](same(11))
	pair[int, int](foreign.Pair(12, 13))
	pair[int, string](foreign.Pair(14, "pair"))
	read[int](foreign.Constrained(15))
	pointer, field := foreign.Pointer(16)
	p := pointer.(*struct{ X int })
	p.X = 17
	fmt.Println(p.X, *field, p == pointer.(*struct{ X int }))
	nested[int](foreign.Nested(18))
	nested[string](foreign.Nested("nested"))

	read[int](create(7))
	x, ok := create(8).(struct{ X int })
	fmt.Println(x.X, ok)
	read[int](foreign.Create(9))
	y, ok := foreign.Create("text").(struct{ X string })
	fmt.Println(y.X, ok)
	z, ok := foreign.Read[int](create(10))
	fmt.Println(z, ok)
}
