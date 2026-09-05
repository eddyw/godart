// The program proves reflect.DeepEqual over a struct that holds a channel, a
// complex number, and a function. Go reports 2 nil functions deeply equal, and
// it reports 2 functions that hold a body not deeply equal.
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

func main() {
	a := new(mixed)
	b := new(mixed)
	fmt.Println(reflect.DeepEqual(a, b), reflect.DeepEqual(*a, *b))

	b.C = 1
	fmt.Println(reflect.DeepEqual(a, b))
	b.C = 0

	b.D = func() {}
	fmt.Println(reflect.DeepEqual(a, b))
	b.D = nil

	b.B = complex(1, 2)
	fmt.Println(reflect.DeepEqual(a, b))
	b.B = 0

	b.A = make(chan int)
	fmt.Println(reflect.DeepEqual(a, b))

	var v any = new(mixed)
	fmt.Println(reflect.DeepEqual(v, reflect.New(reflect.TypeOf(mixed{})).Interface()))

	// The same shape with no name. TestUnmarshal of encoding/json reads
	// exactly this pair: the table holds new(shape), and the test builds a
	// second one with reflect.New of the element type.
	shape := new(struct {
		A chan int
		B complex128
		C int
		D func()
	})
	fresh := reflect.New(reflect.TypeOf(shape).Elem())
	fmt.Println(reflect.DeepEqual(shape, fresh.Interface()))
	fmt.Println(fresh.Elem().IsZero(), reflect.ValueOf(shape).Elem().IsZero())
	fmt.Printf("%T %T\n", shape, fresh.Interface())
	shape.C = 3
	fmt.Println(reflect.DeepEqual(shape, fresh.Interface()))

	// A table of the package holds the pointer, exactly as the table of
	// TestUnmarshal of encoding/json does. The literal writes the zero value
	// of each field, and reflect.New reads the zero of the descriptor.
	tableType := reflect.TypeOf(table[0].ptr).Elem()
	fmt.Println(reflect.DeepEqual(table[0].ptr, reflect.New(tableType).Interface()))

	orig := reflect.ValueOf(table[0].ptr).Elem()
	made := reflect.New(tableType).Elem()
	for i := 0; i < orig.NumField(); i++ {
		fmt.Println(i, reflect.DeepEqual(orig.Field(i).Interface(), made.Field(i).Interface()))
	}

	var c1 chan int
	c2 := table[0].ptr.(*struct {
		A chan int
		B complex128
		C int
		D func()
	}).A
	fmt.Println(c1 == c2, c1 == nil, c2 == nil)
}

var table = []struct {
	ptr any
}{
	{ptr: &struct {
		A chan int
		B complex128
		C int
		D func()
	}{}},
}
