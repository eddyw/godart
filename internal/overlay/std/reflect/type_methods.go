//go:build dart

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package reflect

import "iter"

type fieldScan struct {
	typ   Type
	index []int
}

func (t *rtype) FieldByNameFunc(match func(string) bool) (result StructField, ok bool) {
	if t.Kind() != Struct {
		panic("reflect: FieldByNameFunc of non-struct type " + t.String())
	}
	next := []fieldScan{{typ: t}}
	var nextCount map[Type]int
	visited := map[Type]bool{}
	for len(next) > 0 {
		current, count := next, nextCount
		next = nil
		nextCount = map[Type]int{}
		for _, scan := range current {
			if visited[scan.typ] {
				continue
			}
			visited[scan.typ] = true
			for i := 0; i < scan.typ.NumField(); i++ {
				field := scan.typ.Field(i)
				if match(field.Name) {
					if count[scan.typ] > 1 || ok {
						return StructField{}, false
					}
					result = field
					result.Index = append(append([]int(nil), scan.index...), i)
					ok = true
					continue
				}
				if ok || !field.Anonymous {
					continue
				}
				typ := field.Type
				if typ.Kind() == Pointer {
					typ = typ.Elem()
				}
				if typ.Kind() != Struct {
					continue
				}
				if nextCount[typ] > 0 {
					nextCount[typ] = 2
					continue
				}
				nextCount[typ] = 1
				if count[scan.typ] > 1 {
					nextCount[typ] = 2
				}
				index := append(append([]int(nil), scan.index...), i)
				next = append(next, fieldScan{typ, index})
			}
		}
		if ok {
			return
		}
	}
	return
}

func (t *rtype) Fields() iter.Seq[StructField] {
	if t.Kind() != Struct {
		panic("reflect: Fields of non-struct type " + t.String())
	}
	return func(yield func(StructField) bool) {
		for i := range t.NumField() {
			if !yield(t.Field(i)) {
				return
			}
		}
	}
}

func (t *rtype) Methods() iter.Seq[Method] {
	return func(yield func(Method) bool) {
		for i := range t.NumMethod() {
			if !yield(t.Method(i)) {
				return
			}
		}
	}
}

func (t *rtype) Ins() iter.Seq[Type] {
	if t.Kind() != Func {
		panic("reflect: Ins of non-func type " + t.String())
	}
	return func(yield func(Type) bool) {
		for i := range t.NumIn() {
			if !yield(t.In(i)) {
				return
			}
		}
	}
}

func (t *rtype) Outs() iter.Seq[Type] {
	if t.Kind() != Func {
		panic("reflect: Outs of non-func type " + t.String())
	}
	return func(yield func(Type) bool) {
		for i := range t.NumOut() {
			if !yield(t.Out(i)) {
				return
			}
		}
	}
}

func (t *rtype) OverflowComplex(x complex128) bool {
	k := t.Kind()
	switch k {
	case Complex64:
		return overflowFloat32(real(x)) || overflowFloat32(imag(x))
	case Complex128:
		return false
	}
	panic("reflect: OverflowComplex of non-complex type " + t.String())
}

func (t *rtype) OverflowFloat(x float64) bool {
	k := t.Kind()
	switch k {
	case Float32:
		return overflowFloat32(x)
	case Float64:
		return false
	}
	panic("reflect: OverflowFloat of non-float type " + t.String())
}

func (t *rtype) OverflowInt(x int64) bool {
	k := t.Kind()
	switch k {
	case Int, Int8, Int16, Int32, Int64:
		bitSize := t.Size() * 8
		trunc := (x << (64 - bitSize)) >> (64 - bitSize)
		return x != trunc
	}
	panic("reflect: OverflowInt of non-int type " + t.String())
}

func (t *rtype) OverflowUint(x uint64) bool {
	k := t.Kind()
	switch k {
	case Uint, Uintptr, Uint8, Uint16, Uint32, Uint64:
		bitSize := t.Size() * 8
		trunc := (x << (64 - bitSize)) >> (64 - bitSize)
		return x != trunc
	}
	panic("reflect: OverflowUint of non-uint type " + t.String())
}
