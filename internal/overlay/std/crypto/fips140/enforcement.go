//go:build dart

// This file patches the file enforcement.go of the package crypto/fips140.
//
// The real file reads 3 functions of the runtime through a go:linkname
// directive, and each 1 holds the bypass of the enforcement in the goroutine.
// The GODEBUG setting fips140 always holds its default under gart, thus the
// strict mode never turns on and no bypass is ever necessary.
package fips140

import "internal/godebug"

// WithoutEnforcement runs f. The strict FIPS 140-3 mode never turns on under
// gart, thus the call performs no other operation.
func WithoutEnforcement(f func()) {
	f()
}

var enabled = godebug.New("fips140").Value() == "only"

// Enforced reports whether the strict FIPS 140-3 mode is on.
func Enforced() bool {
	return enabled
}
