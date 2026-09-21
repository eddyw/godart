package main

import (
	"fmt"
	"reflect"
	"unsafe"
)

type inner[T any] struct {
	Pad   byte
	Value T
}
type outer[T any] struct {
	Pad byte
	inner[T]
	Tail struct{}
}

func dimensions[T any](value T) {
	var x outer[T]
	var p *outer[T]
	typ := reflect.TypeFor[T]()
	fmt.Println(unsafe.Sizeof(value), unsafe.Alignof(value), typ.Size(), typ.Align())
	fmt.Println(unsafe.Sizeof(x), unsafe.Alignof(x), unsafe.Offsetof(x.Value), unsafe.Offsetof(p.Tail))
	fmt.Println(reflect.TypeFor[outer[T]]().Size(), reflect.TypeFor[outer[T]]().Align())
	fmt.Println(unsafe.Sizeof(func() T { panic("size operand") }()), unsafe.Alignof(func() T { panic("align operand") }()))
	fmt.Println(unsafe.Sizeof(struct {
		P unsafe.Pointer
		V T
	}{V: value}), unsafe.Alignof(struct {
		P unsafe.Pointer
		V T
	}{V: value}))
	var ch chan T
	fmt.Println(unsafe.Sizeof(<-ch), unsafe.Alignof(<-ch))
}

//gart:sync
type metadata func() uintptr

func syncSize[T any]() uintptr {
	var ch chan T
	f := metadata(func() uintptr { return unsafe.Sizeof(<-ch) })
	return f()
}
func syncAlign[T any](get func() T) uintptr {
	f := metadata(func() uintptr { return unsafe.Alignof(get()) })
	return f()
}
func main() {
	fmt.Println(syncSize[int64](), syncAlign[complex64](nil))
	dimensions(true)
	dimensions(int8(1))
	dimensions(int16(1))
	dimensions(int32(1))
	dimensions(int64(1))
	dimensions(float32(1))
	dimensions(float64(1))
	dimensions(complex64(1))
	dimensions(complex128(1))
	dimensions("x")
	dimensions([3]int16{})
	dimensions([0]int64{})
	dimensions(struct{}{})
	dimensions([]byte(nil))
	dimensions(map[string]int(nil))
	dimensions((chan int)(nil))
	dimensions((func())(nil))
	dimensions((*int)(nil))
	dimensions[any](nil)
}
