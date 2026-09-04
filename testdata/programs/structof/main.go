package main

import (
	"fmt"
	"reflect"
)

// pair names the shape that the first case builds, thus the program holds a
// class of it.
type pair struct {
	A int    `json:"a"`
	B string `json:"b"`
}

func main() {
	// A shape that the program names gives the type of that shape.
	named := reflect.StructOf([]reflect.StructField{
		{Name: "A", Type: reflect.TypeFor[int](), Tag: `json:"a"`},
		{Name: "B", Type: reflect.TypeFor[string](), Tag: `json:"b"`},
	})
	fmt.Println("text:", named.String())
	fmt.Println("kind:", named.Kind(), "fields:", named.NumField())
	fmt.Println("same as pair:", named == reflect.TypeFor[pair]())

	v := reflect.New(named).Elem()
	v.Field(0).SetInt(7)
	v.Field(1).SetString("ok")
	fmt.Println("read:", v.Field(0).Int(), v.Field(1).String())
	fmt.Println("tag:", named.Field(0).Tag.Get("json"), named.Field(1).Tag.Get("json"))

	// A shape that no code names still gives each answer of a Type.
	fresh := reflect.StructOf([]reflect.StructField{
		{Name: "Q", Type: reflect.TypeFor[bool]()},
		{Name: "R", Type: reflect.TypeFor[float64]()},
	})
	fmt.Println("fresh:", fresh.String(), fresh.NumField(), fresh.Kind())
	fmt.Println("field names:", fresh.Field(0).Name, fresh.Field(1).Name)
	fmt.Println("field types:", fresh.Field(0).Type, fresh.Field(1).Type)

	fv := reflect.New(fresh).Elem()
	fv.Field(0).SetBool(true)
	fv.Field(1).SetFloat(2.5)
	fmt.Println("fresh read:", fv.Field(0).Bool(), fv.Field(1).Float())

	// The empty struct.
	empty := reflect.StructOf(nil)
	fmt.Println("empty:", empty.String(), empty.NumField())
}
