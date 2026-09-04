//go:build dart

// This file patches the file xor.go of the package
// crypto/internal/fips140/subtle.
//
// The real file calls xorBytes with the address of the first byte of each
// slice, and an assembly file or the package unsafe holds that function. This
// file holds the loop over the 3 slices instead.
package subtle

import "crypto/internal/fips140/alias"

// XORBytes writes dst[i] = x[i] ^ y[i] for each i below n, where n is the
// smaller of the length of x and of y. It gives n.
//
// It panics when the length of dst is below n, and it writes nothing then.
//
// dst and x, and dst and y, must overlap exactly or not at all.
func XORBytes(dst, x, y []byte) int {
	n := min(len(x), len(y))
	if n == 0 {
		return 0
	}
	if n > len(dst) {
		panic("subtle.XORBytes: dst too short")
	}
	if alias.InexactOverlap(dst[:n], x[:n]) || alias.InexactOverlap(dst[:n], y[:n]) {
		panic("subtle.XORBytes: invalid overlap")
	}
	d := dst[:n]
	a := x[:n]
	b := y[:n]
	for i := range d {
		d[i] = a[i] ^ b[i]
	}
	return n
}
