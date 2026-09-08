// The witness of the identity of a pointer into 2 different slices, and of
// reflect.Type.AssignableTo of 2 types that hold 1 underlying type.
package main

import (
	"fmt"
	"reflect"
)

type userBytes []byte

type otherBytes []byte

type userPtr *int

type userMap map[string]int

type userArray [2]int

type userChan chan int

type userInt int

func main() {
	a := []byte("foo")
	b := make([]byte, 3)
	copy(b, a)
	fmt.Println("2 lists", &a[0] == &b[0])
	fmt.Println("1 list", &a[0] == &a[0])
	c := a[1:]
	fmt.Println("view", &a[1] == &c[0])

	var u userBytes
	sv := reflect.ValueOf(a)
	dv := reflect.ValueOf(&u).Elem()
	fmt.Println("assignable", sv.Type().AssignableTo(dv.Type()))
	var src any = a
	switch src.(type) {
	case []byte:
		fmt.Println("case []byte matches")
	default:
		fmt.Println("case []byte misses")
	}
	clone := make([]byte, len(a))
	copy(clone, a)
	dv.Set(reflect.ValueOf(clone))
	fmt.Println("after set", string(u), &u[0] == &a[0], &u[0] == &clone[0])

	fmt.Println("-- the underlying type --")
	report("[]byte to userBytes", reflect.TypeOf(a), reflect.TypeOf(userBytes(nil)))
	report("userBytes to []byte", reflect.TypeOf(userBytes(nil)), reflect.TypeOf(a))
	report("userBytes to otherBytes",
		reflect.TypeOf(userBytes(nil)), reflect.TypeOf(otherBytes(nil)))
	report("[]int to userBytes", reflect.TypeOf([]int{1}), reflect.TypeOf(userBytes(nil)))
	report("*int to userPtr", reflect.TypeOf(new(int)), reflect.TypeOf(userPtr(nil)))
	report("map to userMap",
		reflect.TypeOf(map[string]int{}), reflect.TypeOf(userMap(nil)))
	report("array to userArray",
		reflect.TypeOf([2]int{}), reflect.TypeOf(userArray{}))
	report("array of 3 to userArray",
		reflect.TypeOf([3]int{}), reflect.TypeOf(userArray{}))
	report("chan to userChan",
		reflect.TypeOf(make(chan int)), reflect.TypeOf(userChan(nil)))
	report("recv chan to userChan",
		reflect.TypeOf(make(<-chan int)), reflect.TypeOf(userChan(nil)))
	// A conversion between 2 named struct types is outside the subset, thus
	// ConvertibleTo of the 2 gives false and the 2 answers differ from Go.
	// docs/parity.md holds the gap.
	report("int to userInt", reflect.TypeOf(1), reflect.TypeOf(userInt(1)))
}

func report(name string, from, to reflect.Type) {
	fmt.Println(name, from.AssignableTo(to), from.ConvertibleTo(to))
}
