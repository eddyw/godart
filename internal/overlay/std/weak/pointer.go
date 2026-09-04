//go:build dart

// Package weak provides ways to safely reference memory weakly, that is,
// without preventing its reclamation.
//
// The Dart form of a weak pointer is a WeakReference of dart:core. The time of
// the reclamation follows the collector of Dart, exactly as the time of a
// cleanup does. Read docs/parity.md.
package weak

import "syscall/dart"

// Pointer is a weak pointer to a value of type T.
//
// Two Pointer values always compare equal if the pointers from which they were
// created compare equal. The zero value of a Pointer is nil, and Value gives
// nil for it.
type Pointer[T any] struct {
	// Mention T in the type definition to prevent conversions
	// between Pointer types, like we do for sync/atomic.Pointer.
	_ [0]*T

	ref *dart.Object
}

// Make creates a weak pointer from a pointer to some value of type T.
func Make[T any](ptr *T) Pointer[T] {
	if ptr == nil {
		return Pointer[T]{}
	}
	return Pointer[T]{ref: dart.WeakRef(ptr)}
}

// Value returns the original pointer used to create the weak pointer.
// It returns nil if the value pointed to by the original pointer was reclaimed
// by the garbage collector.
func (p Pointer[T]) Value() *T {
	if p.ref == nil {
		return nil
	}
	v := dart.WeakTarget(p.ref)
	if v == nil {
		return nil
	}
	return v.(*T)
}
