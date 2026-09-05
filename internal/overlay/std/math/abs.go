//go:build dart

// This file patches the file abs.go of the package math.
//
// The real file reads the bit pattern and it clears the sign bit. Each read of
// that pattern crosses a buffer of the Dart runtime, thus the interop gives
// the same result with 1 operation.
package math

import "syscall/dart"

// Abs returns the absolute value of x.
//
// Special cases are:
//
//	Abs(±Inf) = +Inf
//	Abs(NaN) = NaN
func Abs(x float64) float64 {
	return dart.Abs(x)
}
