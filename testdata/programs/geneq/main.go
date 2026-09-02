package main

import "fmt"

type pair[T, U any] struct {
	Name T
	Data U
}

type plain struct {
	Name string
	Data int
}

type wrap[T comparable] struct{ V T }

func main() {
	a := pair[string, int]{Name: "name", Data: 1}
	b := pair[string, int]{Name: "name", Data: 1}
	c := pair[string, int]{Name: "name", Data: 2}
	fmt.Println("equal:", a == b, "not equal:", a == c)

	p := plain{Name: "name", Data: 1}
	q := plain{Name: "name", Data: 1}
	fmt.Println("plain:", p == q)

	fmt.Println("pointer:", &a == &b, "self:", &a == &a)

	w1 := wrap[string]{V: "x"}
	w2 := wrap[string]{V: "x"}
	fmt.Println("wrap:", w1 == w2)

	m := map[pair[string, int]]int{a: 7}
	fmt.Println("map read:", m[b])

	list := []pair[string, int]{a, c}
	fmt.Println("in list:", list[0] == b)
}
