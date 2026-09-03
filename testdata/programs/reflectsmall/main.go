// The program reads the predicates and the small names of the package reflect.
package main

import (
	"fmt"
	"reflect"
	"sort"
)

type inner struct {
	B int
	C string
}

type outer struct {
	inner
	A int
	D []int
}

type dup1 struct{ N int }
type dup2 struct{ N int }
type both struct {
	dup1
	dup2
}

type greeter struct{ Name string }

func (g greeter) Hello(who string) string { return g.Name + " greets " + who }

func (g greeter) Count() int { return len(g.Name) }

func main() {
	// The 4 Can predicates.
	fmt.Println(reflect.ValueOf(1).CanInt(), reflect.ValueOf(uint(1)).CanInt())
	fmt.Println(reflect.ValueOf(uint8(1)).CanUint(), reflect.ValueOf(1).CanUint())
	fmt.Println(reflect.ValueOf(1.5).CanFloat(), reflect.ValueOf(1).CanFloat())
	fmt.Println(reflect.ValueOf(1+2i).CanComplex(), reflect.ValueOf(1.5).CanComplex())

	// Complex and SetComplex.
	c := reflect.ValueOf(3 + 4i)
	fmt.Println(c.Complex(), c.Kind())
	var cell complex128
	pc := reflect.ValueOf(&cell).Elem()
	pc.SetComplex(5 + 6i)
	fmt.Println(cell)

	// Comparable.
	fmt.Println(reflect.ValueOf(1).Comparable(), reflect.ValueOf([]int{1}).Comparable())
	fmt.Println(reflect.ValueOf(outer{}).Comparable(), reflect.ValueOf(inner{}).Comparable())
	fmt.Println(reflect.ValueOf([2]int{}).Comparable(), reflect.Value{}.Comparable())

	// The 4 Overflow names.
	v8 := reflect.ValueOf(int8(0))
	fmt.Println(v8.OverflowInt(127), v8.OverflowInt(128))
	u8 := reflect.ValueOf(uint8(0))
	fmt.Println(u8.OverflowUint(255), u8.OverflowUint(256))
	f32 := reflect.ValueOf(float32(0))
	fmt.Println(f32.OverflowFloat(1e38), f32.OverflowFloat(1e39))
	c64 := reflect.ValueOf(complex64(0))
	fmt.Println(c64.OverflowComplex(1e38), c64.OverflowComplex(1e39))
	fmt.Println(reflect.ValueOf(0).OverflowInt(1<<62), reflect.ValueOf(0.0).OverflowFloat(1e300))

	// ValueError.
	err := &reflect.ValueError{Method: "reflect.Value.Int", Kind: reflect.String}
	fmt.Println(err.Error(), err.Kind, err.Method)
	fmt.Println((&reflect.ValueError{Method: "reflect.Value.Len"}).Error())

	// Clear of a map and of a slice.
	m := map[string]int{"a": 1, "b": 2}
	reflect.ValueOf(m).Clear()
	fmt.Println(len(m))
	s := []int{1, 2, 3}
	reflect.ValueOf(s).Clear()
	fmt.Println(s)

	// MakeMapWithSize.
	mm := reflect.MakeMapWithSize(reflect.TypeOf(m), 8)
	mm.SetMapIndex(reflect.ValueOf("k"), reflect.ValueOf(9))
	fmt.Println(mm.Interface().(map[string]int))

	// VisibleFields walks each embedded struct.
	names := []string{}
	for _, f := range reflect.VisibleFields(reflect.TypeOf(outer{})) {
		names = append(names, fmt.Sprint(f.Name, f.Index))
	}
	fmt.Println(names)
	dups := []string{}
	for _, f := range reflect.VisibleFields(reflect.TypeOf(both{})) {
		dups = append(dups, f.Name)
	}
	fmt.Println(dups)

	// Swapper exchanges 2 elements.
	list := []string{"a", "b", "c"}
	swap := reflect.Swapper(list)
	swap(0, 2)
	fmt.Println(list)
	sort.Slice(list, func(i, j int) bool { return list[i] < list[j] })
	fmt.Println(list)

	// FieldByNameFunc reads a promoted field.
	ov := reflect.ValueOf(outer{inner: inner{B: 4, C: "c"}, A: 1})
	fmt.Println(ov.FieldByNameFunc(func(n string) bool { return n == "B" }).Int())
	fmt.Println(ov.FieldByNameFunc(func(n string) bool { return n == "Z" }).IsValid())

	// CallSlice reads a variadic function.
	sum := func(head int, rest ...int) int {
		out := head
		for _, x := range rest {
			out += x
		}
		return out
	}
	fv := reflect.ValueOf(sum)
	out := fv.CallSlice([]reflect.Value{reflect.ValueOf(1), reflect.ValueOf([]int{2, 3})})
	fmt.Println(out[0].Int())

	// SetIterKey and SetIterValue.
	src := map[string]int{"one": 1}
	var key string
	var val int
	it := reflect.ValueOf(src).MapRange()
	for it.Next() {
		reflect.ValueOf(&key).Elem().SetIterKey(it)
		reflect.ValueOf(&val).Elem().SetIterValue(it)
	}
	fmt.Println(key, val)

	// The 3 fields of Method.
	gt := reflect.TypeOf(greeter{})
	for i := 0; i < gt.NumMethod(); i++ {
		me := gt.Method(i)
		fmt.Println(me.Name, me.Type, me.PkgPath, me.IsExported(), me.Index)
	}

	// Fields and Methods give a sequence.
	for f, fv := range reflect.ValueOf(outer{A: 7}).Fields() {
		fmt.Println("field", f.Name, fv.Kind())
	}
	for me, mv := range reflect.ValueOf(greeter{Name: "ann"}).Methods() {
		fmt.Println("method", me.Name, mv.Kind())
	}

	// SetCap and SetLen.
	buf := make([]int, 4, 8)
	bv := reflect.ValueOf(&buf).Elem()
	bv.SetCap(6)
	fmt.Println(len(buf), cap(buf))
	bv.SetLen(2)
	fmt.Println(len(buf), cap(buf))
}
