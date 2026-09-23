package main

import (
	"fmt"
	"reflect"
)

type Cell struct {
	N int
	A [2]int
}

func extract[T any](value any) (T, bool) { x, ok := value.(T); return x, ok }
func mutate[T any](original any) {
	value, ok := original.(T)
	v := reflect.ValueOf(&value).Elem()
	if v.Kind() == reflect.Struct {
		v.Field(0).SetInt(79)
	} else {
		v.Index(0).SetInt(83)
	}
	fmt.Println("generic copy", original, value, ok)
}

func main() {
	cell := Cell{7, [2]int{11, 13}}
	v := reflect.ValueOf(&cell).Elem()
	captured := v.Interface()
	cell.N = 17
	cell.A[0] = 19
	fmt.Println("snapshot", captured)
	original := any(Cell{23, [2]int{29, 31}})
	single := original.(Cell)
	tuple, ok := original.(Cell)
	single.N = 37
	tuple.N = 41
	tuple.A[0] = 43
	fmt.Println("assert", original, single, tuple, ok)
	generic, ok := extract[Cell](original)
	generic.N = 47
	generic.A[1] = 53
	fmt.Println("generic", original, generic, ok)
	ref, ok := reflect.TypeAssert[Cell](v)
	ref.N = 59
	ref.A[1] = 61
	fmt.Println("type assert", cell, ref, ok)
	array := [2]int{67, 71}
	arrayView := reflect.ValueOf(&array).Elem()
	arrayCaptured := arrayView.Interface()
	array[0] = 73
	fmt.Println("array", arrayCaptured, array)
	mutate[Cell](original)
	mutate[[2]int](arrayCaptured)
	arrayTuple, ok := arrayCaptured.([2]int)
	arrayTuple[0] = 89
	fmt.Println("array assert", arrayCaptured, arrayTuple, ok)
	pointer := any(&cell)
	pointerTuple, ok := pointer.(*Cell)
	pointerTuple.N = 97
	fmt.Println("pointer", cell.N, ok)
	missing, ok := original.([2]int)
	fmt.Println("mismatch", missing, ok)
	var nilValue any
	missingCell, ok := nilValue.(Cell)
	fmt.Println("nil", missingCell, ok)
}
