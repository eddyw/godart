//go:build dart

// The crypto initialization rejects FIPS mode before this package can run.
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
