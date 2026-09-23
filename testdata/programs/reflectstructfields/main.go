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
}
