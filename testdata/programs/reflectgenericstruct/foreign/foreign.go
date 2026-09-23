package foreign

import (
	"fmt"
	"reflect"
)

var Early = reflect.Zero(reflect.StructOf([]reflect.StructField{{Name: "X", Type: reflect.TypeFor[bool]()}})).Interface()

func Private[T any]() {
	want := reflect.TypeFor[struct{ x T }]()
	got := reflect.StructOf([]reflect.StructField{want.Field(0)})
	value, ok := reflect.Zero(got).Interface().(struct{ x T })
	fmt.Println("foreign private", want == got, ok, value.x)
}
