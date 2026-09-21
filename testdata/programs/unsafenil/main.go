package main

import (
	"fmt"
	"reflect"
	"unsafe"
)

type Pointer unsafe.Pointer

func zero[T ~unsafe.Pointer]() T { var value T; return value }

func main() {
	var p unsafe.Pointer
	q := unsafe.Pointer(uintptr(0))
	n := unsafe.Pointer(nil)
	var named Pointer
	if p != q || p != n || p != nil || named != nil || zero[Pointer]() != nil {
		panic("nil pointer conversion")
	}
	values := map[unsafe.Pointer]int{p: 7}
	if values[q] != 7 {
		panic("nil pointer key")
	}
	var boxed any = p
	if boxed == nil || boxed.(unsafe.Pointer) != nil || any(q) != boxed {
		panic("typed nil interface")
	}
	r := reflect.ValueOf(p)
	if r.Kind() != reflect.UnsafePointer || !r.IsNil() || !r.IsZero() || r.Pointer() != 0 {
		panic("nil pointer reflection")
	}
	if reflect.Zero(r.Type()).Interface().(unsafe.Pointer) != nil {
		panic("reflected zero")
	}
	fmt.Printf("%v %p %T %v\n", p, p, p, named)
	fmt.Println("ok")
}
