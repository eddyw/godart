//go:build dart

// This file patches the file floor_asm.go of the package math.
//
// The real file declares each function with no body, because assembly holds
// the code. The portable Go form reads Modf, which reads the bit pattern
// through a buffer of the Dart runtime. Dart holds 1 method for each of the 3
// operations, thus the flag stands true and the caller takes this path.
//
// Each Dart method follows the rule of Go: a value that is not finite passes
// through, and a negative zero keeps its sign.
package math

import "syscall/dart"

const haveArchFloor = true

func archFloor(x float64) float64 {
	return dart.Floor(x)
}

const haveArchCeil = true

func archCeil(x float64) float64 {
	return dart.Ceil(x)
}

const haveArchTrunc = true

func archTrunc(x float64) float64 {
	return dart.Trunc(x)
}
