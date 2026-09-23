package main

import (
	"fmt"
	"reflect"
)

func nodes[T comparable](value T) {
	type node struct {
		Value T
		Next  *node
	}
	first := node{Value: value}
	first.Next = &first
	copy := first
	fmt.Println(copy.Value == value, copy.Next == &first, copy.Next.Next == &first)
	pointer := reflect.New(reflect.TypeFor[node]())
	pointer.Elem().Set(reflect.ValueOf(first))
	other := pointer.Interface().(*node)
	fmt.Println(other.Value == value, other.Next == &first)
	var holder any = first
	asserted := holder.(node)
	fmt.Println(asserted.Value == value, asserted.Next == &first)
	type view node
	converted := (*view)(&first)
	fmt.Println(converted.Next == &first)
	converted.Next = nil
	fmt.Println(first.Next == nil)
	list := []node{copy}
	array := [1]node{copy}
	table := map[int]node{0: copy}
	ch := make(chan node, 1)
	ch <- copy
	fmt.Println(list[0].Next == &first, array[0].Next == &first, table[0].Next == &first, (<-ch).Next == &first)
	fmt.Println(reflect.DeepEqual(copy, asserted))
}
func main() { nodes(42); nodes("text") }
