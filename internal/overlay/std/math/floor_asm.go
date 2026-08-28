//go:build dart

// This file patches the file floor_asm.go of the package math.
//
// The real file declares each function with no body, because assembly
// holds the code. This file holds the pure Go form of the same package,
// thus the flag stays false and the caller takes the portable path.
package math

const haveArchFloor = false

func archFloor(x float64) float64 {
	panic("not implemented")
}

const haveArchCeil = false

func archCeil(x float64) float64 {
	panic("not implemented")
}

const haveArchTrunc = false

func archTrunc(x float64) float64 {
	panic("not implemented")
}
