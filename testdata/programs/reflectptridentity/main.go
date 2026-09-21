package main

import (
	"fmt"
	"reflect"
	"weak"
)

func verify[T any](value T) {
	s := []T{value, value, value, value}
	direct := &s[2]
	addresses := []*T{
		reflect.ValueOf(s).Index(2).Addr().Interface().(*T),
		reflect.ValueOf(s[1:]).Index(1).Addr().Interface().(*T),
		reflect.ValueOf(s).Slice(1, 4).Index(1).Addr().Interface().(*T),
		reflect.ValueOf((*[3]T)(s[1:])).Elem().Index(1).Addr().Interface().(*T),
	}
	for _, p := range addresses {
		if p != direct {
			panic("reflected address differs")
		}
		if reflect.ValueOf(p).Pointer() != reflect.ValueOf(direct).Pointer() {
			panic("reflected pointer identity differs")
		}
		if weak.Make(p) != weak.Make(direct) {
			panic("reflected weak address differs")
		}
	}
	if reflect.ValueOf(s).Index(1).Addr().Interface().(*T) == direct {
		panic("distinct addresses match")
	}
	a := [4]T{value, value, value, value}
	if reflect.ValueOf(&a).Elem().Index(2).Addr().Interface().(*T) != &a[2] {
		panic("array address differs")
	}
	replacement := new(T)
	*addresses[0] = *replacement
	if !reflect.DeepEqual(s[2], *replacement) {
		panic("reflected pointer write")
	}
	fmt.Println("equal reflected addresses")
}
func main() {
	verify(7)
	verify(true)
	verify(3.5)
	verify(complex(1.0, 2.0))
	verify("text")
	verify([]int{7})
	verify(map[int]int{1: 7})
	verify(make(chan int))
	verify(new(int))
	verify[any](7)
}
