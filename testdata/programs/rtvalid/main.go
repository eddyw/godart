// Command rtvalid proves the validity mark of a reflect.Value.
//
// A nil interface, a nil pointer, and a nil map each read as nil, thus the
// value alone cannot answer IsValid. Go holds an entry of a map that carries a
// nil interface valid, and it gives the zero Value for a key that the map does
// not hold.
package main

import (
	"fmt"
	"reflect"
)

type Row struct {
	Any  any
	Ptr  *int
	Map  map[string]int
	List []int
	Text string
}

func mark(name string, v reflect.Value) {
	kind := "invalid"
	if v.IsValid() {
		kind = v.Kind().String()
	}
	fmt.Println(name, v.IsValid(), kind)
}

func main() {
	// A map of an interface element.
	m := map[string]any{"nil": nil, "one": 1}
	mv := reflect.ValueOf(m)
	mark("map/nil", mv.MapIndex(reflect.ValueOf("nil")))
	mark("map/one", mv.MapIndex(reflect.ValueOf("one")))
	mark("map/absent", mv.MapIndex(reflect.ValueOf("absent")))

	// A map of a pointer element.
	pm := map[string]*int{"nil": nil}
	pv := reflect.ValueOf(pm)
	mark("ptrmap/nil", pv.MapIndex(reflect.ValueOf("nil")))
	mark("ptrmap/absent", pv.MapIndex(reflect.ValueOf("absent")))

	// The zero Value of each kind.
	mark("zero/any", reflect.Zero(reflect.TypeOf((*any)(nil)).Elem()))
	mark("zero/ptr", reflect.Zero(reflect.TypeOf((*int)(nil))))
	mark("zero/map", reflect.Zero(reflect.TypeOf(map[string]int{})))
	mark("zero/slice", reflect.Zero(reflect.TypeOf([]int{})))
	mark("valueof/nil", reflect.ValueOf(nil))

	// A field that holds nil.
	r := Row{}
	rv := reflect.ValueOf(r)
	mark("field/any", rv.Field(0))
	mark("field/ptr", rv.Field(1))
	mark("field/map", rv.Field(2))
	mark("field/list", rv.Field(3))
	mark("field/text", rv.Field(4))

	// An element of a slice of interface.
	list := []any{nil, 2}
	lv := reflect.ValueOf(list)
	mark("index/nil", lv.Index(0))
	mark("index/two", lv.Index(1))

	// Elem of a nil pointer and of a nil interface.
	var np *Row
	mark("elem/nilptr", reflect.ValueOf(np).Elem())
	var ni any
	mark("elem/nilany", reflect.ValueOf(&ni).Elem().Elem())

	// A name that no field and no method holds.
	mark("field/absent", rv.FieldByName("Absent"))
	mark("method/absent", rv.MethodByName("Absent"))

	// A write of a nil interface into a map keeps the key, and a write of the
	// zero Value drops it.
	out := reflect.MakeMap(reflect.TypeOf(map[string]any{}))
	out.SetMapIndex(reflect.ValueOf("keep"), reflect.Zero(reflect.TypeOf((*any)(nil)).Elem()))
	out.SetMapIndex(reflect.ValueOf("gone"), reflect.ValueOf(3))
	out.SetMapIndex(reflect.ValueOf("gone"), reflect.Value{})
	got := out.Interface().(map[string]any)
	_, keep := got["keep"]
	_, gone := got["gone"]
	fmt.Println("setmapindex", len(got), keep, gone)

	// The iterator of a map reads the same entries.
	it := mv.MapRange()
	count := 0
	valid := 0
	for it.Next() {
		count++
		if it.Value().IsValid() {
			valid++
		}
	}
	fmt.Println("range", count, valid)
}
