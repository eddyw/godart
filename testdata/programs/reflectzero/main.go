// The program proves reflect.Value.IsZero over each kind. Go walks a struct
// and an array that hold no comparison, and it answers a channel, a function,
// a map, and a slice with IsNil.
package main

import (
	"fmt"
	"reflect"
)

type mixed struct {
	A chan int
	B complex128
	C int
	D func()
}

type plain struct {
	X int
	Y string
}

type holder struct {
	M map[string]int
	S []int
	P *int
	I any
}

func main() {
	var m mixed
	fmt.Println(reflect.ValueOf(m).IsZero(), reflect.ValueOf(&m).Elem().IsZero())
	m.C = 1
	fmt.Println(reflect.ValueOf(m).IsZero())
	m.C = 0
	m.D = func() {}
	fmt.Println(reflect.ValueOf(m).IsZero())

	var p plain
	fmt.Println(reflect.ValueOf(p).IsZero())
	p.Y = "x"
	fmt.Println(reflect.ValueOf(p).IsZero())

	var h holder
	fmt.Println(reflect.ValueOf(h).IsZero())
	h.S = []int{}
	fmt.Println(reflect.ValueOf(h).IsZero())

	var funcs [2]func()
	fmt.Println(reflect.ValueOf(funcs).IsZero())
	funcs[1] = func() {}
	fmt.Println(reflect.ValueOf(funcs).IsZero())

	var nums [3]int
	fmt.Println(reflect.ValueOf(nums).IsZero())
	nums[2] = 5
	fmt.Println(reflect.ValueOf(nums).IsZero())

	var ch chan int
	var fn func()
	var mp map[string]int
	var sl []int
	var pt *int
	var iface any
	fmt.Println(reflect.ValueOf(&ch).Elem().IsZero(),
		reflect.ValueOf(&fn).Elem().IsZero(),
		reflect.ValueOf(&mp).Elem().IsZero(),
		reflect.ValueOf(&sl).Elem().IsZero(),
		reflect.ValueOf(&pt).Elem().IsZero(),
		reflect.ValueOf(&iface).Elem().IsZero())

	fmt.Println(reflect.ValueOf(0).IsZero(), reflect.ValueOf(1).IsZero(),
		reflect.ValueOf("").IsZero(), reflect.ValueOf("a").IsZero(),
		reflect.ValueOf(0.0).IsZero(), reflect.ValueOf(1.5).IsZero(),
		reflect.ValueOf(false).IsZero(), reflect.ValueOf(true).IsZero(),
		reflect.ValueOf(complex(0, 0)).IsZero(), reflect.ValueOf(complex(0, 1)).IsZero())
}
