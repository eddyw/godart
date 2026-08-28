//go:build dart

// Package reflect gives the part of the standard reflect package that the
// surface holds.
//
// The compiler cannot read the real source. That source reads the memory layout
// of a value, and the Dart VM gives no such access. The type descriptor of the
// runtime library holds the name, the kind, and each field of a type, thus the
// package gives a deep comparison and the description of a type.
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

// A Type describes a Go type.
type Type struct {
	text       string
	kind       Kind
	comparable bool
}

// TypeOf gives the type of the value inside an interface.
func TypeOf(v any) Type {
	text, comparable := dart.TypeInfo(v)
	return Type{text: text, kind: Kind(dart.Kind(v)), comparable: comparable}
}

// String gives the text of the type. It holds the name of the package, and not
// the import path.
func (t Type) String() string { return t.text }

// Kind gives the class of the type.
func (t Type) Kind() Kind { return t.kind }

// Comparable reports whether Go compares 2 values of the type with ==.
func (t Type) Comparable() bool { return t.comparable }
