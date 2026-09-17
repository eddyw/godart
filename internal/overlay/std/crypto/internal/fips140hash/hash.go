//go:build dart

// This file patches the file hash.go of the package
// crypto/internal/fips140hash.
//
// The real file reads the inner digest of a crypto/sha3.SHA3 through a
// go:linkname directive, because the field of that struct is not exported.
// This file gives h itself instead.
//
// The HMAC of Go reads the hash through the interface hash.Hash alone, and
// crypto/sha3.SHA3 holds each method of the inner digest, thus the bytes of the
// result stay the bytes of Go. The type reaches 2 other tests: the service
// indicator of the FIPS 140-3 mode, and ApprovedHash of the FIPS 140-only mode.
// Neither mode ever turns on under gart.
package fips140hash

import (
	// Preserve the registration side effect of the upstream dependency.
	_ "crypto/sha3"
	"hash"
)

// Unwrap gives h.
func Unwrap(h hash.Hash) hash.Hash {
	return h
}

// UnwrapNew gives a function that calls newHash and applies Unwrap to the
// result.
func UnwrapNew[Hash hash.Hash](newHash func() Hash) func() hash.Hash {
	return func() hash.Hash { return Unwrap(newHash()) }
}
