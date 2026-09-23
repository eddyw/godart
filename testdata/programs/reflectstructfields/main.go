package main

import (
	"fmt"
	"reflect"
)

func describe(fields []reflect.StructField) {
	defer func() {
		if value := recover(); value != nil {
			fmt.Println(value)
		}
	}()
	fmt.Println(reflect.StructOf(fields))
}

func main() {
	integer := reflect.TypeOf(0)
	describe(nil)
	describe([]reflect.StructField{{Name: "_", PkgPath: "p", Type: integer}, {Name: "_", PkgPath: "p", Type: integer}})
	describe([]reflect.StructField{{Name: "X", Type: integer}, {Name: "X", Type: integer}})
	describe([]reflect.StructField{{Name: "X", Type: integer}, {Name: "Missing"}})
	describe([]reflect.StructField{{Type: integer}})
	describe([]reflect.StructField{{Name: "X", Type: integer}, {Name: "1Field", Type: integer}})
	describe([]reflect.StructField{{Name: "Φ١", Type: integer}})
	describe([]reflect.StructField{{Name: "١Field", Type: integer}})
	describe([]reflect.StructField{{Name: "\xff", Type: integer}})
	type inner struct {
		Before int
		Leaf   string
	}
	type outer struct {
		Padding bool
		Nested  inner
	}
	typ := reflect.TypeOf(outer{})
	for _, index := range [][]int{nil, {}} {
		field := typ.FieldByIndex(index)
		fmt.Println(field.Type == typ, field.Name == "", field.Index == nil)
		fmt.Println(reflect.DeepEqual(field, reflect.StructField{Type: typ}))
	}
	index := []int{1, 1}
	field := typ.FieldByIndex(index)
	fmt.Println(field.Name, field.Type, field.Index)
	index[1] = 0
	fmt.Println(field.Index)
	func() {
		defer func() { fmt.Println(recover()) }()
		integer.FieldByIndex(nil)
	}()

}
