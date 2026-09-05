//go:build dart

// This file patches the file sqrt.go of the package math.
//
// The compiler of Go replaces a call of math.sqrt with 1 machine instruction,
// and gart holds no such rule. The real file thus gives a loop of 54 steps
// that builds the result bit by bit, and 1 square root costs about 140
// nanoseconds. dart:math gives the same instruction, and IEEE 754 names 1
// correct result for each input, thus the 2 forms agree bit for bit.
package math

import "syscall/dart"

// Sqrt returns the square root of x.
//
// Special cases are:
//
//	Sqrt(+Inf) = +Inf
//	Sqrt(±0) = ±0
//	Sqrt(x < 0) = NaN
//	Sqrt(NaN) = NaN
func Sqrt(x float64) float64 {
	return sqrt(x)
}

func sqrt(x float64) float64 {
	return dart.Sqrt(x)
}
