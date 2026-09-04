// The program proves that a local generic type of a generic function holds the
// identity of each type argument, of the outer one and of its own.
package main

import (
	"fmt"
	"reflect"
)

type intish interface{ ~int }

type Int int

var seen []reflect.Type

func F[A intish]() {
	type S struct{ x int }
	type T[B intish] struct{}
	type U[B intish] struct{ inner T[B] }

	seen = append(seen,
		reflect.TypeOf(S{}),
		reflect.TypeOf(T[int]{}),
		reflect.TypeOf(T[Int]{}),
		reflect.TypeOf(U[Int]{}),
		reflect.TypeOf(&S{}))
}

func main() {
	F[int]()
	F[Int]()

	for i, a := range seen {
		for j, b := range seen[:i+1] {
			if a == b {
				fmt.Printf("%d and %d name 1 type: %v\n", j, i, a)
			}
		}
	}
	fmt.Println("count", len(seen))
}
