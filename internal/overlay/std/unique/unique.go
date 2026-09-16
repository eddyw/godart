//go:build dart

// Package unique gives 1 handle for each value of equal content.
//
// Go frees a handle that no code holds, through a cleanup of its runtime. The
// subset holds no such cleanup, thus the map of this overlay keeps each value
// for the life of the program. docs/parity.md holds that gap.
package unique

import "syscall/dart"

// A Handle names 1 value. 2 handles are equal when the 2 values are equal.
type Handle[T comparable] struct {
	value *T
}

// Value gives a copy of the value that built the handle.
func (h Handle[T]) Value() T { return *h.value }

// The static type distinguishes Make[any](x) from Make[T](x).
type handleKey struct {
	typeID int
	value  any
}

var handles = map[handleKey]any{}

// Make gives the handle of the value. 2 calls with equal values give equal
// handles.
func Make[T comparable](value T) Handle[T] {
	key := handleKey{dart.DescOf((*T)(nil)), value}
	if held, ok := handles[key]; ok {
		return Handle[T]{value: held.(*T)}
	}
	v := new(T)
	*v = value
	handles[key] = v
	return Handle[T]{value: v}
}
