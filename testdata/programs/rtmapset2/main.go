// Command rtmapset2 proves that a write of a struct through reflect copies each
// level.
//
// Go copies a struct and an array at an assignment. The Dart form of each 1 is
// an object, thus a write of the reference alone would join the 2 values. A
// caller that reuses 1 cell for each entry of a map thus gave each key the same
// nested object.
package main

import (
	"fmt"
	"reflect"
)

// Deep holds 3 levels of struct, plus an array of struct.
type Deep struct {
	Name  string
	Inner Mid
	Cells [2]Leaf
}

type Mid struct {
	Tag  string
	Leaf Leaf
}

type Leaf struct {
	N int
}

func main() {
	// 1 cell serves each entry, exactly as the decoder of encoding/json does.
	m := reflect.MakeMap(reflect.TypeOf(map[string]Deep{}))
	cell := reflect.New(reflect.TypeOf(Deep{})).Elem()
	for i, key := range []string{"a", "b", "c"} {
		cell.Field(0).SetString(key)
		cell.Field(1).Field(0).SetString("tag-" + key)
		cell.Field(1).Field(1).Field(0).SetInt(int64(i + 1))
		cell.Field(2).Index(0).Field(0).SetInt(int64(10 * (i + 1)))
		cell.Field(2).Index(1).Field(0).SetInt(int64(100 * (i + 1)))
		m.SetMapIndex(reflect.ValueOf(key), cell)
	}
	out := m.Interface().(map[string]Deep)
	for _, key := range []string{"a", "b", "c"} {
		v := out[key]
		fmt.Printf("%s %s %s %d %d %d\n", key, v.Name, v.Inner.Tag,
			v.Inner.Leaf.N, v.Cells[0].N, v.Cells[1].N)
	}

	// A write of the whole struct into a pointer also copies each level.
	src := Deep{
		Name:  "src",
		Inner: Mid{Tag: "t", Leaf: Leaf{N: 7}},
		Cells: [2]Leaf{{N: 8}, {N: 9}},
	}
	dst := reflect.New(reflect.TypeOf(Deep{})).Elem()
	dst.Set(reflect.ValueOf(src))
	src.Inner.Leaf.N = 700
	src.Cells[0].N = 800
	got := dst.Interface().(Deep)
	fmt.Println("ptrset", got.Inner.Leaf.N, got.Cells[0].N, src.Inner.Leaf.N)

	// A write into a field copies each level too.
	holder := reflect.New(reflect.TypeOf(Deep{})).Elem()
	mid := Mid{Tag: "m", Leaf: Leaf{N: 21}}
	holder.Field(1).Set(reflect.ValueOf(mid))
	mid.Leaf.N = 2100
	fmt.Println("fieldset", holder.Interface().(Deep).Inner.Leaf.N, mid.Leaf.N)

	// A write into an element of a slice copies each level.
	list := reflect.MakeSlice(reflect.TypeOf([]Mid{}), 2, 2)
	list.Index(0).Set(reflect.ValueOf(mid))
	list.Index(1).Set(reflect.ValueOf(mid))
	mid.Leaf.N = 4200
	rows := list.Interface().([]Mid)
	fmt.Println("indexset", rows[0].Leaf.N, rows[1].Leaf.N, mid.Leaf.N)

	// An array value in a map also holds a copy of its own.
	am := reflect.MakeMap(reflect.TypeOf(map[string][2]Leaf{}))
	acell := reflect.New(reflect.TypeOf([2]Leaf{})).Elem()
	for i, key := range []string{"x", "y"} {
		acell.Index(0).Field(0).SetInt(int64(i + 1))
		acell.Index(1).Field(0).SetInt(int64(i + 5))
		am.SetMapIndex(reflect.ValueOf(key), acell)
	}
	amap := am.Interface().(map[string][2]Leaf)
	fmt.Println("array", amap["x"][0].N, amap["x"][1].N, amap["y"][0].N, amap["y"][1].N)

	// The 2 entries of the map must hold 2 objects, thus a write of 1 leaves
	// the other alone.
	pm := map[string]Deep{}
	pm["one"] = out["a"]
	pm["two"] = out["b"]
	fmt.Println("keep", pm["one"].Inner.Leaf.N, pm["two"].Inner.Leaf.N)
}
