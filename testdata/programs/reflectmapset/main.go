// The program proves that reflect.Value.SetMapIndex writes a copy of a struct
// value. Go copies the value into the map, thus a caller that reuses 1 cell
// for each entry gives each key its own value. The map decoder of
// encoding/json/v2 reuses exactly 1 cell.
package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
)

type small struct{ Tag string }

func main() {
	m := map[string]small{}
	mv := reflect.ValueOf(m)
	// cell holds 1 value for each entry, exactly as the decoder of v2 does.
	cell := reflect.New(reflect.TypeOf(small{})).Elem()
	for _, tag := range []string{"a", "b", "c"} {
		cell.Field(0).SetString("tag" + tag)
		mv.SetMapIndex(reflect.ValueOf(tag), cell)
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Println(k, m[k].Tag)
	}

	// The same shape reaches a program through encoding/json.
	var out map[string]small
	if err := json.Unmarshal([]byte(`{"17":{"Tag":"tag17"},"18":{"Tag":"tag18"}}`), &out); err != nil {
		fmt.Println("unmarshal:", err)
		return
	}
	fmt.Println(out["17"].Tag, out["18"].Tag)

	// A nil interface writes the entry, and the key stays. The zero Value
	// removes the entry.
	var any1 map[string]any
	if err := json.Unmarshal([]byte(`{"T":null}`), &any1); err != nil {
		fmt.Println("unmarshal:", err)
		return
	}
	held, ok := any1["T"]
	fmt.Println(len(any1), held == nil, ok)

	plain := map[string]any{"k": 1}
	av := reflect.ValueOf(plain)
	av.SetMapIndex(reflect.ValueOf("n"), reflect.Zero(reflect.TypeOf((*any)(nil)).Elem()))
	av.SetMapIndex(reflect.ValueOf("k"), reflect.Value{})
	nilValue, holdsN := plain["n"]
	_, holdsK := plain["k"]
	fmt.Println(len(plain), nilValue == nil, holdsN, holdsK)

	// A map of a pointer keeps the reference, exactly as Go does.
	pm := map[string]*small{}
	pv := reflect.ValueOf(pm)
	one := &small{Tag: "shared"}
	pv.SetMapIndex(reflect.ValueOf("x"), reflect.ValueOf(one))
	pv.SetMapIndex(reflect.ValueOf("y"), reflect.ValueOf(one))
	one.Tag = "moved"
	fmt.Println(pm["x"].Tag, pm["y"].Tag, pm["x"] == pm["y"])
}
