package main

import (
	"fmt"
	"reflect"
	"unsafe"
)

var calls int

func sideEffect() int { calls++; return 0 }

func capacity[T any](name string, n int) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(name, "panic", err)
		}
	}()
	c := make(chan T, n)
	fmt.Println(name, cap(c))
}

func dynamic(name string, elem reflect.Type, n int) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(name, "panic", err)
		}
	}()
	c := reflect.MakeChan(reflect.ChanOf(reflect.BothDir, elem), n)
	fmt.Println(name, c.Cap())
}

func main() {
	fmt.Println("metadata", unsafe.Sizeof(sideEffect()), calls, unsafe.Sizeof(unsafe.Pointer(nil)))
	capacity[int]("ordinary", 3)
	capacity[int]("negative", -1)
	capacity[int]("large", 1<<59)
	capacity[[65535]byte]("array", 1<<46)
	capacity[struct {
		X int
		Y struct{}
	}]("padding", 1<<59)
	capacity[struct{}]("empty", 1<<59)
	capacity[[0]int]("zero-array", 1<<59)
	dynamic("reflect-int", reflect.TypeFor[int](), 1<<59)
	dynamic("reflect-empty", reflect.TypeFor[struct{}](), 1<<59)
}
