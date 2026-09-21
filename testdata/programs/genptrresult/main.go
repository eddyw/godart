package main

import "fmt"

type record struct{ Value int }
type integerPointer *int
type recordPointer *record
type arrayPointer *[2]int

func fresh[P ~*T, T any](value T) P     { return &value }
func pair[P ~*T, T any](value T) (P, P) { return &value, &value }
func assign[P ~*T, T any](value T) P {
	var pointer P
	pointer = &value
	return pointer
}
func accept[P ~*T, T any](pointer P) P     { return pointer }
func argument[P ~*T, T any](value T) P     { return accept[P](&value) }
func concrete[P ~*T, T any](pointer P) *T  { return pointer }
func scalar[P ~*int](value int) P          { return &value }
func structure[P ~*record](value record) P { return &value }
func array[P ~*[2]int](value [2]int) P     { return &value }

func verify[P ~*T, T comparable](original, changed T) {
	a := fresh[P](original)
	b := assign[P](original)
	c := argument[P](original)
	left, right := pair[P](original)
	if a == b || a == c || left != right {
		panic("pointer identity")
	}
	*left = changed
	if *right != changed || *a != original || *b != original || *c != original {
		panic("pointer value")
	}
	unwrapped := concrete(a)
	*unwrapped = changed
	if *a != changed {
		panic("pointer alias")
	}
	var empty P
	if concrete(empty) != nil {
		panic("nil pointer")
	}
	fmt.Println(*a, *b, *c, *left, *right)
}

func main() {
	verify[*int](11, 17)
	verify[integerPointer](23, 29)
	verify[*record](record{31}, record{37})
	verify[recordPointer](record{41}, record{43})
	verify[*[2]int]([2]int{47, 53}, [2]int{59, 61})
	verify[arrayPointer]([2]int{67, 71}, [2]int{73, 79})
	fmt.Println(*scalar[integerPointer](83), *structure[recordPointer](record{89}), *array[arrayPointer]([2]int{97, 101}))
}
