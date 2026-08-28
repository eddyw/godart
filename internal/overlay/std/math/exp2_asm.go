//go:build dart

// This file patches the file exp2_asm.go of the package math.
//
// The real file declares each function with no body, because assembly
// holds the code. This file holds the pure Go form of the same package,
// thus the flag stays false and the caller takes the portable path.
package math

const haveArchExp2 = false

func archExp2(x float64) float64 {
	panic("not implemented")
}
