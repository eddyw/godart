// The program proves that a conversion of a reflect.Value to an interface
// type keeps the dynamic type of the value.
//
// Go holds the interface as the static type of the location alone. Value.Type
// gives the interface, and Value.Interface gives the concrete value, thus a
// type assertion to the concrete type holds.
//
// MarshalFunc of encoding/json/v2 reads that shape: it converts the address of
// the value to the type parameter, and the body of the marshaler asserts the
// concrete type of the result.
package main

import (
	"encoding"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

func row(label string, value any) {
	fmt.Printf("%-34s %v\n", label, value)
}

type nocaseString string

func (k nocaseString) MarshalText() ([]byte, error) {
	return []byte(strings.ToLower(string(k))), nil
}

func (k *nocaseString) UnmarshalText(b []byte) error {
	*k = nocaseString(strings.ToLower(string(b)))
	return nil
}

type shape interface{ area() int }

type square struct{ side int }

func (s square) area() int { return s.side * s.side }

var (
	textType  = reflect.TypeFor[encoding.TextMarshaler]()
	shapeType = reflect.TypeFor[shape]()
	anyType   = reflect.TypeFor[any]()
	sortType  = reflect.TypeFor[sort.Interface]()
)

// castTo holds the shape of addressableValue.castTo of encoding/json/v2.
func castTo(v reflect.Value, to reflect.Type) reflect.Value {
	switch to.Kind() {
	case reflect.Interface:
		return v.Addr().Convert(to)
	case reflect.Pointer:
		return v.Addr()
	default:
		return v
	}
}

func main() {
	k := nocaseString("Hello")
	pointer := reflect.ValueOf(&k)
	conv := pointer.Convert(textType)
	row("kind of the conversion", conv.Kind())
	row("type of the conversion", conv.Type())
	row("type inside it", reflect.TypeOf(conv.Interface()))

	got, ok := reflect.TypeAssert[encoding.TextMarshaler](conv)
	row("the assert of the interface", ok)
	concrete, ok := got.(*nocaseString)
	row("the assert of the pointer", fmt.Sprintf("%v %v", ok, concrete != nil))
	text, _ := got.MarshalText()
	row("the text of the marshaler", string(text))

	// A value that stands in a slot, exactly as json/v2 holds it.
	holder := reflect.New(reflect.TypeFor[nocaseString]()).Elem()
	holder.SetString("WORLD")
	cast := castTo(holder, textType)
	row("the cast of a slot", cast.Type())
	viaCast, _ := reflect.TypeAssert[encoding.TextMarshaler](cast)
	body, _ := viaCast.MarshalText()
	row("the text of the cast", body)

	// A value type that holds the method on the value.
	s := square{side: 3}
	sv := reflect.ValueOf(s).Convert(shapeType)
	row("the shape type", sv.Type())
	row("the shape inside", reflect.TypeOf(sv.Interface()))
	shp, ok := reflect.TypeAssert[shape](sv)
	row("the assert of the shape", fmt.Sprintf("%v %d", ok, shp.area()))

	// The empty interface takes each value.
	av := reflect.ValueOf(42).Convert(anyType)
	row("the empty interface", fmt.Sprintf("%v %v %v", av.Kind(), av.Type(), reflect.TypeOf(av.Interface())))

	// An interface to a wider interface.
	wide := sv.Convert(anyType)
	row("the wider interface", fmt.Sprintf("%v %v", wide.Type(), reflect.TypeOf(wide.Interface())))

	// A conversion that no method set allows.
	row("a shape is no sort.Interface", reflect.TypeOf(s).ConvertibleTo(sortType))

	// A nil pointer keeps its type inside the interface.
	var nilKey *nocaseString
	nv := reflect.ValueOf(nilKey).Convert(textType)
	row("a nil pointer inside", fmt.Sprintf("%v %v %v", nv.Type(), nv.IsNil(), reflect.TypeOf(nv.Interface())))

	// A conversion of a value that already holds an interface type.
	var box any = s
	bv := reflect.ValueOf(&box).Elem()
	row("a slot of any", fmt.Sprintf("%v %v", bv.Kind(), bv.Type()))
	row("the value inside the slot", reflect.TypeOf(bv.Interface()))
}
