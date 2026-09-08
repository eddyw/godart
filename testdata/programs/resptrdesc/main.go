// The program proves 2 rules of a type that the package reflect builds at run
// time.
//
// A description of a pointer that PointerTo builds must name the Dart type of
// the value. A cell of Object? fits no typed location: the setter of a field
// of the type `**count` casts to a pointer of a pointer of an integer.
//
// A conversion between 2 defined types of 1 basic kind must hold, because Go
// converts between 2 types whose underlying types are identical.
//
// encoding/json/v2 reads both: makePointerArshaler allocates the element of a
// field type, and unmarshalEmbeddedFallbackNext converts a string to a defined
// string type.
package main

import (
	"fmt"
	"reflect"
)

func row(label string, value any) {
	fmt.Printf("%-40s %v\n", label, value)
}

type count int

type flag bool

type name string

type holder struct {
	Number **count
	Mark   **flag
	Text   **name
}

func main() {
	// PointerTo runs first, thus the description of *count comes from the
	// constructor of the run time and not from the emitted table.
	pcount := reflect.PointerTo(reflect.TypeFor[count]())
	pflag := reflect.PointerTo(reflect.TypeFor[flag]())
	pname := reflect.PointerTo(reflect.TypeFor[name]())
	row("the pointer types", fmt.Sprintf("%v %v %v", pcount, pflag, pname))

	var h holder
	v := reflect.ValueOf(&h).Elem()
	v.Field(0).Set(reflect.New(pcount))
	v.Field(1).Set(reflect.New(pflag))
	v.Field(2).Set(reflect.New(pname))
	row("each field holds a pointer", fmt.Sprintf("%v %v %v",
		h.Number != nil, h.Mark != nil, h.Text != nil))

	inner := reflect.New(reflect.TypeFor[count]())
	inner.Elem().SetInt(5)
	v.Field(0).Elem().Set(inner)
	row("the value behind 2 pointers", **h.Number)

	// A conversion between 2 defined types of 1 basic kind.
	row("string to name", reflect.ValueOf("hello").Convert(reflect.TypeFor[name]()).Interface())
	row("name to string", reflect.ValueOf(name("hi")).Convert(reflect.TypeFor[string]()).Interface())
	row("bool to flag", reflect.ValueOf(true).Convert(reflect.TypeFor[flag]()).Interface())
	row("flag to bool", reflect.ValueOf(flag(false)).Convert(reflect.TypeFor[bool]()).Interface())
	row("int to count", reflect.ValueOf(7).Convert(reflect.TypeFor[count]()).Interface())
	row("string is convertible to name",
		reflect.TypeFor[string]().ConvertibleTo(reflect.TypeFor[name]()))
	row("bool is no name",
		reflect.TypeFor[flag]().ConvertibleTo(reflect.TypeFor[name]()))
}
