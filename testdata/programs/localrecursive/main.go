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
	fmt.Println(pointer.Elem().FieldByName("Next").Elem().FieldByName("Value").Interface() == value)
	fmt.Println(pointer.Elem().FieldByName("Next").Elem().FieldByName("Next").Elem().Type() == reflect.TypeFor[node]())
	typ := reflect.TypeFor[node]()
	next := typ.Field(1).Type.Elem()
	fmt.Println(next == typ, next.NumField(), next.Field(0).Type == reflect.TypeFor[T]())
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
