// The program proves that a local type and a package level type of the same
// name hold 2 identities, and that each type built from them also does.
//
// The identity of a type at run time is the name of its descriptor. A local
// type carries the position of its declaration, and a slice, a map, a pointer,
// or a channel of such a type needs that position too. The package
// encoding/json_test holds `type Animal int` at the top and `type Animal
// struct{...}` inside an example, and ExampleUnmarshal decoded into a slice of
// the second one.
package main

import (
	"encoding/json"
	"fmt"
	"reflect"
)

func row(label string, value any) {
	fmt.Printf("%-38s %v\n", label, value)
}

// Animal at the top of the package holds an integer.
type Animal int

func topLevel() {
	var list []Animal
	row("the top level slice", reflect.TypeOf(list))
	row("the top level element", reflect.TypeOf(list).Elem().Kind())
	m := map[string]Animal{}
	row("the top level map", reflect.TypeOf(m))
	row("the zero of the top level", reflect.MakeSlice(reflect.TypeOf(list), 2, 4).Interface())
}

func local() {
	// Animal of this body holds a struct, and it shares the printed name.
	type Animal struct {
		Name  string
		Order string
	}
	var list []Animal
	row("the local slice", reflect.TypeOf(list))
	row("the local element", reflect.TypeOf(list).Elem().Kind())
	m := map[string]Animal{}
	row("the local map", reflect.TypeOf(m))
	grown := reflect.MakeSlice(reflect.TypeOf(list), 1, 2)
	row("the make of the local slice", grown.Interface())

	// Grow reads the descriptor of the slice, and it writes through a pointer
	// of the element type of that descriptor.
	v := reflect.ValueOf(&list).Elem()
	v.Grow(4)
	row("the grow of the local slice", fmt.Sprintf("%d %v", v.Len(), v.Cap() >= 4))

	blob := []byte(`[{"Name":"Platypus","Order":"Monotremata"},{"Name":"Quoll","Order":"Dasyuromorphia"}]`)
	if err := json.Unmarshal(blob, &list); err != nil {
		row("the unmarshal of the local slice", err)
		return
	}
	row("the unmarshal of the local slice", list)
}

func second() {
	// A second body holds a third Animal, thus 3 types share 1 name.
	type Animal struct {
		Legs int
	}
	var list []Animal
	list = append(list, Animal{Legs: 4})
	row("the second local slice", reflect.TypeOf(list))
	row("the second local value", list)
	var box any = list
	_, ok := box.([]Animal)
	row("the assert of the second slice", ok)
	data, err := json.Marshal(list)
	row("the marshal of the second slice", fmt.Sprintf("%s %v", data, err))
}

func main() {
	topLevel()
	local()
	second()
}
