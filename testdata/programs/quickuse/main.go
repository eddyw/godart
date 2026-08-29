package main

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing/quick"
)

type pair struct {
	A int
	B string
}

func reverse(s []int) []int {
	out := make([]int, len(s))
	for i, v := range s {
		out[len(s)-1-i] = v
	}
	return out
}

func main() {
	// A property that holds.
	if err := quick.Check(func(a, b int) bool { return a+b == b+a }, nil); err != nil {
		fmt.Println("add:", err)
	} else {
		fmt.Println("add holds")
	}

	// A property that fails.
	err := quick.Check(func(a int) bool { return a == 0 }, &quick.Config{MaxCount: 200})
	fmt.Println("zero fails:", err != nil)

	// A property over a slice.
	if err := quick.Check(func(s []int) bool {
		return reflect.DeepEqual(reverse(reverse(s)), s)
	}, nil); err != nil {
		fmt.Println("reverse:", err)
	} else {
		fmt.Println("reverse holds")
	}

	// A value of each kind.
	r := rand.New(rand.NewSource(7))
	for _, t := range []reflect.Type{
		reflect.TypeOf(0),
		reflect.TypeOf(""),
		reflect.TypeOf(true),
		reflect.TypeOf(1.5),
		reflect.TypeOf([]int(nil)),
		reflect.TypeOf(map[string]int(nil)),
		reflect.TypeOf(pair{}),
		reflect.TypeOf([3]byte{}),
	} {
		v, ok := quick.Value(t, r)
		fmt.Println(t.Kind(), ok, v.IsValid(), v.Type().Kind() == t.Kind())
	}

	// A call through reflect.
	fn := reflect.ValueOf(func(a int, b string) (string, int) {
		return b + "!", a * 2
	})
	out := fn.Call([]reflect.Value{reflect.ValueOf(3), reflect.ValueOf("go")})
	fmt.Println(len(out), out[0].String(), out[1].Int())
	fmt.Println(fn.Type().NumIn(), fn.Type().NumOut(),
		fn.Type().In(1).Kind(), fn.Type().Out(0).Kind())

	// CheckEqual compares 2 functions.
	if err := quick.CheckEqual(
		func(a int) int { return a * 2 },
		func(a int) int { return a + a }, nil); err != nil {
		fmt.Println("equal:", err)
	} else {
		fmt.Println("equal holds")
	}
}
