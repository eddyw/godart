// The program proves the reflect names that the work of 2026-09-04 added:
// SliceHeader and StringHeader.
package main

import (
	"fmt"
	"reflect"
)

func main() {
	// The 2 header types hold plain numbers, and a read gives the value that
	// the program wrote.
	slice := reflect.SliceHeader{Data: 0x40, Len: 3, Cap: 8}
	fmt.Println("slice:", slice.Data, slice.Len, slice.Cap)
	text := reflect.StringHeader{Data: 0x40, Len: 5}
	fmt.Println("string:", text.Data, text.Len)

	head := reflect.TypeOf(slice)
	fmt.Println("type:", head.Name(), head.NumField(), head.Field(1).Name)

	// A header is a plain value, thus a copy and an equality each work.
	same := slice
	fmt.Println("copy:", same == slice, same.Len)
	same.Len = 9
	fmt.Println("write:", slice.Len, same.Len)

	// The 2 types reach a reflect Value, exactly as each other struct does.
	v := reflect.ValueOf(&text).Elem()
	v.Field(1).SetInt(11)
	fmt.Println("set:", text.Len, v.Type().Field(0).Name)
}
