package main

import (
	"fmt"

	"example.com/promoted/store"
)

type namer interface{ Name() string }

func main() {
	i := store.NewItem("alpha", 3)
	fmt.Println(i.Name(), i.Bump(), i.Bump(), i.Label())
	fmt.Println(i.Key, i.Tag)

	h := store.NewHolder("beta")
	fmt.Println(h.Name(), h.Bump(), h.Key)

	var n namer = i
	fmt.Println("through an interface:", n.Name())

	fmt.Println(store.Inside(i))

	// A method value and a method expression each read the promoted name.
	f := i.Name
	g := (*store.Item).Bump
	fmt.Println(f(), g(i))
}
