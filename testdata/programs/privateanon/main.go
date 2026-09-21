package main

import (
	"fmt"
	"reflect"

	"example.com/privateanon/a"
	"example.com/privateanon/b"
)

func list[T any](v T) []T { return []T{v} }

func main() {
	v := a.Make(7)
	copy := v
	a.Set(&v, 9)
	fmt.Println(a.Read(v), a.Read(copy), a.Read(list(v)[0]))
	fmt.Println(a.Accept(copy), a.Accept(b.Make(7)))
	fmt.Println(any(copy) == any(a.Make(7)), any(copy) == b.Make(7))
	fmt.Println(a.Read(a.Box{Value: v}.Value), a.Read(a.Nested(11).Inner))
	field := reflect.ValueOf(&v).Elem().Field(0)
	meta := reflect.TypeOf(v).Field(0)
	fmt.Println(field.Int(), field.CanInterface(), field.CanSet(), meta.PkgPath)
	fmt.Printf("%+v\n", a.Nested(11))
}
