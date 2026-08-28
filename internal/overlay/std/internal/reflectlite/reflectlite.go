//go:build dart

// Package reflectlite gives the part of reflect that the standard library
// reads. The package context reads the text of a type and the comparable flag.
package reflectlite

import "syscall/dart"

// A Type describes a Go type.
type Type struct {
	text       string
	comparable bool
}

// TypeOf gives the type of the value inside an interface.
func TypeOf(v any) Type {
	text, comparable := dart.TypeInfo(v)
	return Type{text: text, comparable: comparable}
}

// String gives the text of the type. It holds the name of the package, and not
// the import path.
func (t Type) String() string { return t.text }

// Comparable reports whether Go compares 2 values of the type.
func (t Type) Comparable() bool { return t.comparable }
