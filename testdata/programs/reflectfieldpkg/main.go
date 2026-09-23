package main

import (
	"example.com/reflectfieldpkg/model"
	"fmt"
	"reflect"
)

type Alias = int

func makeType(name, path string) reflect.Type {
	return reflect.StructOf([]reflect.StructField{{Name: name, PkgPath: path, Type: reflect.TypeOf(0)}})
}
func panicText(f func()) { defer func() { fmt.Println(recover()) }(); f() }
func main() {
	a, b := makeType("x", "one"), makeType("x", "two")
	fmt.Println(a == b, a.Field(0).PkgPath, b.Field(0).PkgPath, a.Field(0).IsExported())
	for _, name := range []string{"X", "Φ", "φ"} {
		t := makeType(name, "")
		f := t.Field(0)
		v := reflect.New(t).Elem().Field(0)
		fmt.Println(name, f.PkgPath, f.IsExported(), v.CanSet(), v.CanInterface())
	}
	upper := makeType("X", "one")
	fmt.Println(upper.Field(0).IsExported(), reflect.New(upper).Elem().Field(0).CanInterface())
	local := makeType("x", "example.com/reflectfieldpkg/model")
	fmt.Println(local == reflect.TypeOf(model.Record{}))
	fmt.Println(model.Read(reflect.Zero(local).Interface().(model.Record)))
	alias := reflect.StructOf([]reflect.StructField{{Name: "Alias", Type: reflect.TypeOf(0), Anonymous: true}})
	fmt.Println(alias == reflect.TypeOf(struct{ Alias }{}), alias.Field(0).Name, alias.Field(0).Anonymous)
	fmt.Println(reflect.Zero(alias).Interface().(struct{ Alias }).Alias)
	panicText(func() { makeType("x", "") })
	panicText(func() { makeType("_", "") })
	panicText(func() {
		reflect.StructOf([]reflect.StructField{{Name: "X", PkgPath: "one", Type: reflect.TypeOf(0), Anonymous: true}})
	})
	panicText(func() {
		reflect.StructOf([]reflect.StructField{{Name: "x", PkgPath: "one", Type: reflect.TypeOf(0)}, {Name: "y", PkgPath: "two", Type: reflect.TypeOf(0)}})
	})
}
