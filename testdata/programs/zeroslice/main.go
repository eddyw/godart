package main

import (
	"fmt"
	"reflect"
	"slices"
)

type Empty struct{ Pad [0]uint64 }
type Wrapped[T any] struct{ Value T }

const maxInt = int(^uint(0) >> 1)

func generic[T any](n int) []T          { return make([]T, n) }
func wrapped[T any](n int) []Wrapped[T] { return make([]Wrapped[T], n) }
func failure(name string, f func())     { defer func() { fmt.Println(name, recover()) }(); f() }

//go:noinline
func indexAt(s []Empty, index int) Empty { return s[index] }

func main() {
	sequence := make([]Empty, maxInt-2, maxInt)
	sequence = slices.Insert(sequence, 1, sequence[3:4]...)
	fmt.Println("insert", len(sequence), cap(sequence))
	sequence = slices.Replace(sequence, 1, 2, sequence[3:5]...)
	fmt.Println("replace", len(sequence), cap(sequence))
	s := make([]Empty, maxInt-1, maxInt)
	fmt.Println("make", len(s), cap(s), s != nil)
	s = append(s, Empty{})
	fmt.Println("append", len(s), cap(s))
	tail := s[maxInt-2 : maxInt-1 : maxInt]
	fmt.Println("slice", len(tail), cap(tail), len(s[maxInt-1:]))
	s[maxInt-1] = Empty{}
	fmt.Println("index", s[maxInt-1] == Empty{})
	pointer := &s[maxInt-1]
	*pointer = Empty{}
	fmt.Println("pointer", pointer != nil, pointer == &s[maxInt-1])
	fmt.Println("copy", copy(s, s[1:]))
	var other []Empty
	other = append(other, s...)
	fmt.Println("spread", len(other), cap(other), other != nil)
	clear(other)
	fmt.Println("deep equal", reflect.DeepEqual(s, other), reflect.DeepEqual(s, s[:maxInt-1]), reflect.DeepEqual([]Empty(nil), make([]Empty, 0)))
	fmt.Println("generic", len(generic[Empty](maxInt)), len(wrapped[Empty](maxInt)))
	arrays := make([][0]int, maxInt)
	ap := &arrays[maxInt-1]
	fmt.Println("array", len(*ap), len(arrays[maxInt-2:]))
	small := make([]Empty, 2, 3)
	small = append(small, Empty{})
	fmt.Println("small", len(small), cap(small), len([2]Empty(small)))
	value := reflect.MakeSlice(reflect.TypeOf([]Empty{}), maxInt, maxInt)
	fmt.Println("reflect make", value.Len(), value.Cap())
	value.Index(maxInt - 1).Set(reflect.ValueOf(Empty{}))
	fmt.Println("reflect index", value.Index(maxInt-1).Interface() == Empty{}, value.Index(maxInt-1).Addr().Interface().(*Empty) != nil)
	part := value.Slice(maxInt-2, maxInt)
	fmt.Println("reflect slice", part.Len(), part.Cap(), reflect.Copy(value, value.Slice(1, maxInt)))
	held := value.Interface().([]Empty)
	reflect.ValueOf(&held).Elem().SetLen(maxInt - 1)
	fmt.Println("reflect setlen", len(held), cap(held))
	dynamic := reflect.MakeSlice(reflect.SliceOf(reflect.StructOf(nil)), maxInt, maxInt)
	fmt.Println("dynamic", dynamic.Len(), dynamic.Cap(), dynamic.Index(maxInt-1).IsZero())
	dynamic.Clear()
	room := make([]Empty, maxInt-2, maxInt-1)
	reflect.ValueOf(&room).Elem().Grow(2)
	fmt.Println("grow", len(room), cap(room))
	ordinary := []int{1, 2, 3, 4}
	window := ordinary[:1:3]
	reflect.ValueOf(&window).Elem().Grow(4)
	fmt.Println("grow retained", window[:3])
	failure("reflect grow", func() { reflect.ValueOf(&s).Elem().Grow(1) })
	failure("reflect spread", func() { _ = reflect.AppendSlice(value, value) })
	reflect.Swapper(s)(maxInt-1, 0)
	failure("swap bound", func() { reflect.Swapper(s)(maxInt, 0) })
	failure("index bound", func() { _ = indexAt(s, maxInt) })
	failure("slice bound", func() { i, j := maxInt, maxInt-1; _ = s[i:j] })
	failure("append bound", func() { _ = append(s, Empty{}) })
	failure("spread bound", func() { _ = append(s, s...) })
	failure("reflect append", func() { _ = reflect.Append(value, reflect.ValueOf(Empty{})) })
}
