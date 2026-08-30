package main

import (
	"fmt"
	"reflect"
)

type empty struct{}

type one struct{ a bool }

type mixed struct {
	a bool
	b int64
	c int8
}

type nested struct {
	m mixed
	s string
}

type trailing struct {
	a int32
	e empty
}

type arrays struct {
	a [3]int16
	b [2]pairs
}

type pairs struct{ x, y int8 }

type wide struct {
	p *int
	s []byte
	i any
	f func()
	m map[string]int
	c chan int
}

func show(v any) {
	t := reflect.TypeOf(v)
	fmt.Printf("%-10s size=%2d align=%d", t.Name(), t.Size(), t.Align())
	if t.Kind() == reflect.Struct {
		for i := 0; i < t.NumField(); i++ {
			fmt.Printf(" %s@%d", t.Field(i).Name, t.Field(i).Offset)
		}
	}
	fmt.Println()
}

func main() {
	show(empty{})
	show(one{})
	show(mixed{})
	show(nested{})
	show(trailing{})
	show(arrays{})
	show(pairs{})
	show(wide{})
	show([4]mixed{})
	show(int32(0))
	show("")
	show([]byte(nil))

	fmt.Println(reflect.TypeOf([0]int64{}).Size(), reflect.TypeOf(empty{}).Align())
	fmt.Println(reflect.TypeOf(mixed{}).FieldAlign(), reflect.TypeOf(one{}).FieldAlign())

	// The name of an instance of a generic type holds each type argument.
	fmt.Printf("%q %q %q\n", reflect.TypeOf([4]mixed{}).Name(),
		reflect.TypeOf(&mixed{}).Name(), reflect.TypeOf(map[string]int{}).Name())
}
