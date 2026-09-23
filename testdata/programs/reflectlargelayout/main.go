package main

import (
	"fmt"
	"reflect"
)

func panicText(f func()) { defer func() { fmt.Println(recover()) }(); f() }
func main() {
	byteType := reflect.TypeFor[byte]()
	emptyType := reflect.TypeFor[struct{}]()
	max := int(^uint(0) >> 1)
	for _, n := range []int{1<<53 - 1, 1<<53 + 1, max} {
		for _, elem := range []reflect.Type{byteType, emptyType} {
			a := reflect.ArrayOf(n, elem)
			fmt.Println(a, a.Len(), a.Size(), a.Align(), a == reflect.ArrayOf(n, elem))
		}
	}
	a := reflect.ArrayOf(max, byteType)
	b := reflect.ArrayOf(max-4, byteType)
	s := reflect.StructOf([]reflect.StructField{{Name: "A", Type: a}, {Name: "B", Type: b}, {Name: "Z", Type: emptyType}})
	fmt.Println(s.Size(), s.Align(), s.Field(1).Offset, s.Field(2).Offset)
	aligned := reflect.StructOf([]reflect.StructField{{Name: "A", Type: a}, {Name: "B", Type: reflect.TypeFor[int32]()}})
	fmt.Println(aligned.Size(), aligned.Align(), aligned.Field(1).Offset)
	panicText(func() { reflect.ArrayOf(max, reflect.TypeFor[uint64]()) })
	panicText(func() { reflect.ArrayOf(max, reflect.ArrayOf(3, byteType)) })
	fmt.Println(reflect.ArrayOf(max, reflect.ArrayOf(2, byteType)).Size())
}
