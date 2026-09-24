package main

import "reflect"

const size = 1<<53 + 1

type Empty struct{}

func same[T comparable](v T) bool { return any(v) == any(v) }

func zero[T any]() [size]T  { var a [size]T; return a }
func pointer() *[size]Empty { println("pointer"); return nil }

func failure(f func()) {
	defer func() { println(recover().(error).Error()) }()
	f()
}

func main() {
	var a [size]Empty
	println(len(a), cap(a))
	a[size-1] = Empty{}
	b := a
	println(a == b, same(a))
	m := map[[size]Empty]int{a: 3}
	println(m[b])
	s := a[size-3:]
	println(len(s), cap(s))
	p := (*[size]Empty)(a[:])
	println(p[size-1] == Empty{})
	c := [size]Empty(a[:])
	println(c == a)
	q := &a[size-1]
	*q = Empty{}
	println(*q == Empty{})
	literal := [size]Empty{size - 1: func() Empty { println("literal"); return Empty{} }()}
	println(literal == a, len(zero[Empty]()), cap(zero[Empty]()))
	println(len(pointer()), cap(pointer()))
	const max = int(^uint(0) >> 1)
	var largest [max]Empty
	largest[max-1] = Empty{}
	duplicate := largest
	println(len(largest), cap(largest), duplicate == largest)
	println(map[[max]Empty]int{largest: 9}[duplicate])
	last := largest[max-2:]
	println(len(last), cap(last))
	typ := reflect.ArrayOf(size, reflect.TypeOf(Empty{}))
	v := reflect.New(typ).Elem()
	v.Set(reflect.ValueOf(a))
	println(v.Len(), v.Cap(), reflect.Copy(v, v))
	whole := v.Slice(0, size)
	println(whole.Len(), whole.Cap())
	converted := whole.Convert(typ)
	println(converted.Interface().([size]Empty) == a)
	convertedPtr := whole.Convert(reflect.PointerTo(typ))
	println(convertedPtr.Interface().(*[size]Empty)[size-1] == Empty{})
	var small [2]Empty
	literalSmall := [2]Empty{{}, {}}
	println(small == literalSmall, any(small) == any(literalSmall))
	println(map[[2]Empty]int{literalSmall: 4}[small])
	println(map[any]int{literalSmall: 5}[small])
	failure(func() { _ = [size]Empty([]Empty{}) })
	failure(func() { _ = (*[size]Empty)([]Empty{}) })
	for i, value := range a {
		println(i, value == Empty{})
		break
	}
	index := size
	failure(func() { println(a[index] == Empty{}) })
	failure(func() { a[index] = Empty{} })
	var nilSlice []Empty
	println((*[0]Empty)(nilSlice) == nil, (*[0]Empty)(a[:0]) == nil)
	var bad [size][0][]int
	failure(func() { v := any(bad); println(v == v) })
	failure(func() { m := map[any]int{}; m[bad] = 1 })
}
