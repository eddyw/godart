//go:build dart

// Package reflect gives the part of the standard reflect package that the
// surface holds.
//
// The compiler cannot read the real source. That source reads the memory layout
// of a value, and the Dart VM gives no such access. The type descriptor of the
// runtime library holds the name, the kind, and each field of a type, thus the
// package gives a deep comparison, the description of a type, and a read and a
// write through a pointer.
//
// `docs/parity.md` lists each gap.
package reflect

import "syscall/dart"

// DeepEqual reports whether 2 values are deeply equal.
//
// 2 values of 2 types are never equal. A slice, a map, a struct, an array, a
// pointer, and an interface each compare their parts, and each other kind
// compares with the equality of Go.
func DeepEqual(x, y any) bool {
	return dart.DeepEqual(x, y)
}

// A Kind names the class of a type, for example `int` or `struct`.
type Kind string

// String gives the name of the kind.
func (k Kind) String() string { return string(k) }

// A Type describes a Go type. Go gives an interface, and the overlay gives a
// struct that holds a handle of the type descriptor of the runtime.
type Type struct {
	desc int
}

// TypeOf gives the type of the value inside an interface.
func TypeOf(v any) Type {
	return Type{desc: dart.DescOf(v)}
}

// String gives the text of the type. It holds the name of the package, and not
// the import path.
func (t Type) String() string {
	text, _, _ := dart.DescInfo(t.desc)
	return text
}

// Kind gives the class of the type.
func (t Type) Kind() Kind {
	_, kind, _ := dart.DescInfo(t.desc)
	return Kind(kind)
}

// Comparable reports whether Go compares 2 values of the type with ==.
func (t Type) Comparable() bool {
	_, _, comparable := dart.DescInfo(t.desc)
	return comparable
}

// Elem gives the element type of a pointer, a slice, an array, a channel, or a
// map. Each other type panics, exactly as in Go.
func (t Type) Elem() Type {
	elem := dart.DescElem(t.desc)
	if elem == 0 {
		panic("reflect: Elem of " + t.String())
	}
	return Type{desc: elem}
}

// A Value holds a Go value. A Value that Elem read also holds the pointer, thus
// Set writes through it.
type Value struct {
	value  any
	holder any
}

// ValueOf gives the value inside an interface.
func ValueOf(v any) Value { return Value{value: v} }

// Zero gives the zero value of a type.
func Zero(t Type) Value { return Value{value: dart.DescZero(t.desc)} }

// Interface gives the value of v inside an interface.
func (v Value) Interface() any { return v.value }

// Elem gives the value behind a pointer. The result holds the pointer, thus Set
// writes through it.
func (v Value) Elem() Value {
	return Value{value: dart.PointerElem(v.value), holder: v.value}
}

// Set writes x into the location that v names. A Value that Elem did not read
// holds no location, thus it panics.
func (v Value) Set(x Value) {
	dart.PointerSet(v.holder, x.value)
}
