//go:build dart

// This file patches the file of the same name of the package sort.
//
// The real file reads internal/reflectlite for the length of a slice of an
// unknown element type and for its swap function. The overlay of reflectlite
// holds the text of a type alone, thus the 3 functions below read the 2
// intrinsics of syscall/dart. Each other line comes from the real file.
package sort

import (
	"math/bits"
	"syscall/dart"
)

// Slice sorts the slice x given the provided less function.
// It panics if x is not a slice.
//
// The sort is not guaranteed to be stable: equal elements
// may be reversed from their original order.
// For a stable sort, use [SliceStable].
//
// The less function must satisfy the same requirements as
// the Interface type's Less method.
//
// Note: in many situations, the newer [slices.SortFunc] function is more
// ergonomic and runs faster.
func Slice(x any, less func(i, j int) bool) {
	length := dart.SliceLen(x)
	swap := func(i, j int) { dart.SliceSwap(x, i, j) }
	limit := bits.Len(uint(length))
	pdqsort_func(lessSwap{less, swap}, 0, length, limit)
}

// SliceStable sorts the slice x using the provided less
// function, keeping equal elements in their original order.
// It panics if x is not a slice.
//
// The less function must satisfy the same requirements as
// the Interface type's Less method.
//
// Note: in many situations, the newer [slices.SortStableFunc] function is more
// ergonomic and runs faster.
func SliceStable(x any, less func(i, j int) bool) {
	length := dart.SliceLen(x)
	swap := func(i, j int) { dart.SliceSwap(x, i, j) }
	stable_func(lessSwap{less, swap}, length)
}

// SliceIsSorted reports whether the slice x is sorted according to the provided less function.
// It panics if x is not a slice.
//
// Note: in many situations, the newer [slices.IsSortedFunc] function is more
// ergonomic and runs faster.
func SliceIsSorted(x any, less func(i, j int) bool) bool {
	n := dart.SliceLen(x)
	for i := n - 1; i > 0; i-- {
		if less(i, i-1) {
			return false
		}
	}
	return true
}
