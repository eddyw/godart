package main

import (
	"fmt"
	"reflect"
)

type hidden interface{ value() int }
type item struct{}

func (item) value() int { return 7 }

func report(call func()) {
	defer func() { fmt.Println(recover()) }()
	call()
	fmt.Println("no panic")
}

func main() {
	report(func() { reflect.TypeAssert[int](reflect.Value{}) })
	report(func() { reflect.TypeAssert[int](reflect.ValueOf(struct{ value int }{7}).Field(0)) })
	var receiver hidden = item{}
	report(func() { reflect.TypeAssert[func() int](reflect.ValueOf(&receiver).Elem().Method(0)) })
}
