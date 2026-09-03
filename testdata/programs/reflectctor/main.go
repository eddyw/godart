// The program reads the 3 type constructors of reflect. Each 1 must build a
// value of the Dart type of the element, thus a type assertion of the result
// reads. The witness is fixedbugs/issue22605 of the test suite of Go.
package main

import (
	"fmt"
	"reflect"
)

type key [8]string

func lookup(m map[[8]string]int) int {
	t := reflect.ArrayOf(8, reflect.TypeOf(""))
	a := reflect.New(t).Elem()
	return m[a.Interface().([8]string)]
}

func main() {
	// issue22605: an array that the constructor builds fits a map key.
	m := map[[8]string]int{}
	m[[8]string{}] = 7
	fmt.Println(lookup(m))

	// A slice that the constructor builds fits its own type.
	st := reflect.SliceOf(reflect.TypeOf(0))
	s := reflect.MakeSlice(st, 2, 3)
	s.Index(0).SetInt(4)
	fmt.Println(s.Interface().([]int), s.Len(), s.Cap())

	// A map that the constructor builds fits its own type.
	mt := reflect.MapOf(reflect.TypeOf(""), reflect.TypeOf(0))
	mv := reflect.MakeMap(mt)
	mv.SetMapIndex(reflect.ValueOf("a"), reflect.ValueOf(1))
	fmt.Println(mv.Interface().(map[string]int))

	// New of a slice type gives a pointer that a type assertion reads.
	p := reflect.New(st)
	fmt.Println(p.Interface().(*[]int) != nil, p.Elem().Interface().([]int) == nil)

	// A named array type keeps its own element type.
	kt := reflect.ArrayOf(8, reflect.TypeOf(""))
	kv := reflect.New(kt).Elem()
	kv.Index(1).SetString("b")
	fmt.Println(kv.Interface().([8]string)[1])

	// A map with an array key compares by value, exactly as Go does. The
	// second key is another object with the same elements.
	am := reflect.MakeMap(reflect.MapOf(kt, reflect.TypeOf(0)))
	am.SetMapIndex(kv, reflect.ValueOf(3))
	twin := reflect.New(kt).Elem()
	twin.Index(1).SetString("b")
	fmt.Println(am.MapIndex(twin).Int(), am.Len())
	am.SetMapIndex(twin, reflect.ValueOf(4))
	fmt.Println(am.Len(), am.MapIndex(kv).Int())

	// Each element of a fresh slice of a struct type holds its own value.
	pt := reflect.TypeOf(pair{})
	ps := reflect.MakeSlice(reflect.SliceOf(pt), 2, 2)
	ps.Index(0).Field(0).SetInt(5)
	fmt.Println(ps.Interface().([]pair))

	// A named array type reads its own name and its element.
	fmt.Println(reflect.TypeOf(key{}), reflect.TypeOf(key{}).Elem())
}

type pair struct{ A, B int }
