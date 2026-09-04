//go:build dart

// This file patches the file constant_time.go of the package
// crypto/internal/constanttime.
//
// boolToUint8 of the real file panics, because the compiler of Go replaces it
// with an instruction of the machine. This file holds the portable form. The
// Dart VM holds no such instruction, thus the branch stands and the time of the
// operation is not constant. Read docs/parity.md.
package constanttime

// Select gives x when v is 1, and y when v is 0.
func Select(v, x, y int) int {
	v = int(boolToUint8(v != 0))
	return ^(v-1)&x | (v-1)&y
}

// ByteEq gives 1 when x is equal to y, and 0 otherwise.
func ByteEq(x, y uint8) int {
	return int(boolToUint8(x == y))
}

// Eq gives 1 when x is equal to y, and 0 otherwise.
func Eq(x, y int32) int {
	return int(boolToUint8(x == y))
}

// LessOrEq gives 1 when x is not above y, and 0 otherwise.
func LessOrEq(x, y int) int {
	return int(boolToUint8(x <= y))
}

func boolToUint8(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}
