//go:build dart

// Package subtle gives the operations of cryptographic code that need a
// constant time.
//
// The real source of Go reads `crypto/internal/constanttime`,
// `crypto/internal/fips140/subtle`, and `internal/runtime/sys`, and each one
// stands outside the surface. `boolToUint8` of the first package panics,
// because the compiler of Go replaces it with an instruction of the machine.
// The overlay holds the same 8 names, and each body is the portable form.
package subtle

// ConstantTimeCompare gives 1 when the 2 slices hold the same bytes, and 0
// otherwise. The time of the call follows the length of the slices, and not
// the content. A pair of slices of different length gives 0 at once.
func ConstantTimeCompare(x, y []byte) int {
	if len(x) != len(y) {
		return 0
	}
	var v byte
	for i := 0; i < len(x); i++ {
		v |= x[i] ^ y[i]
	}
	return ConstantTimeByteEq(v, 0)
}

// ConstantTimeSelect gives x when v is 1, and y when v is 0. Each other value
// of v gives x, because Go holds the same superset behavior.
func ConstantTimeSelect(v, x, y int) int {
	if v != 0 {
		v = 1
	}
	return ^(v-1)&x | (v-1)&y
}

// ConstantTimeByteEq gives 1 when the 2 bytes are equal, and 0 otherwise.
func ConstantTimeByteEq(x, y uint8) int {
	return int((uint32(x^y) - 1) >> 31)
}

// ConstantTimeEq gives 1 when the 2 values are equal, and 0 otherwise.
func ConstantTimeEq(x, y int32) int {
	return int((uint64(uint32(x^y)) - 1) >> 63)
}

// ConstantTimeCopy writes each byte of y into x when v is 1, and it writes
// nothing when v is 0. The 2 slices must hold the same length. Each other
// value of v gives an undefined result.
func ConstantTimeCopy(v int, x, y []byte) {
	if len(x) != len(y) {
		panic("subtle: slices have different lengths")
	}
	xmask := byte(v - 1)
	ymask := byte(^(v - 1))
	for i := 0; i < len(x); i++ {
		x[i] = x[i]&xmask | y[i]&ymask
	}
}

// ConstantTimeLessOrEq gives 1 when x is not greater than y, and 0 otherwise.
// Each value must stand between 0 and 2**31-1.
func ConstantTimeLessOrEq(x, y int) int {
	x32 := int32(x)
	y32 := int32(y)
	return int(((x32 - y32 - 1) >> 31) & 1)
}

// XORBytes writes the exclusive or of x and y into dst, and it gives the count
// of the bytes that it wrote. That count is the length of the shorter of x and
// y. A dst that holds fewer bytes gives a panic, and dst keeps each byte.
//
// Go also panics for a dst that overlaps x or y in part. The check of Go reads
// the package unsafe, thus the overlay holds no such check, and the
// documentation of Go says "may panic". `docs/parity.md` names the difference.
func XORBytes(dst, x, y []byte) int {
	n := min(len(x), len(y))
	if n == 0 {
		return 0
	}
	if n > len(dst) {
		panic("subtle.XORBytes: dst too short")
	}
	for i := 0; i < n; i++ {
		dst[i] = x[i] ^ y[i]
	}
	return n
}

// WithDataIndependentTiming runs f. The mode of the processor that holds the
// timing of each instruction steady needs an instruction of the machine, and
// the Dart target reaches none, thus the call adds nothing. Go behaves the
// same way on a platform that holds no such mode.
func WithDataIndependentTiming(f func()) { f() }
