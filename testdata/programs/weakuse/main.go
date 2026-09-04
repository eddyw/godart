// The program proves that a weak pointer reads its value while a strong name
// holds it, and that 2 weak pointers of 1 value compare equal.
package main

import (
	"fmt"
	"weak"
)

type node struct {
	name  string
	count int
}

func main() {
	var empty weak.Pointer[node]
	fmt.Println("zero value", empty.Value() == nil)

	var none *node
	fmt.Println("nil pointer", weak.Make(none).Value() == nil)

	live := &node{name: "root", count: 1}
	first := weak.Make(live)
	second := weak.Make(live)
	fmt.Println("equal", first == second)

	other := weak.Make(&node{name: "other"})
	fmt.Println("other equal", first == other)

	got := first.Value()
	fmt.Println("same value", got == live)
	got.count = 7
	fmt.Println("write reaches the value", live.count)

	number := 42
	np := weak.Make(&number)
	fmt.Println("number", *np.Value())

	text := "hello"
	tp := weak.Make(&text)
	*tp.Value() = "world"
	fmt.Println("text", text)

	list := []weak.Pointer[node]{first, second, other}
	same := 0
	for _, p := range list {
		if p == first {
			same++
		}
	}
	fmt.Println("count of equal", same)
}
