package main

import (
	"example.com/globalptr/values"
	"fmt"
)

func main() {
	p := &values.Value
	fmt.Println(p == values.Pointer(), p == &values.Value, *p)
	*p = 4
	values.Set(5)
	values.Value.Add(2)
	values.Increment()
	fmt.Println(*p, values.Value)
	list := &values.List
	fmt.Println(list == values.ListPointer())
	*list = []int{9}
	fmt.Println(values.List)
	fmt.Println(&*p == p)
	var nilInt *int
	var nilStruct *struct{ X int }
	caught(func() { _ = &*nilInt })
	caught(func() { _ = &*nilStruct })
}

func caught(f func()) {
	defer func() { fmt.Println(recover()) }()
	f()
}
