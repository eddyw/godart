//go:build dart

// This file patches the file entropy_fips140.go of the package
// crypto/internal/fips140/drbg.
//
// The real file reads crypto/internal/entropy, which measures the jitter of the
// CPU through the package unsafe. Only the FIPS 140-3 mode reaches getEntropy,
// and that mode never turns on under gart, thus no seed of the jitter source is
// ever necessary. Read is a call of crypto/internal/sysrand in that state.
package drbg

func getEntropy() *[SeedSize]byte {
	panic("fips140/drbg: the FIPS 140-3 mode never turns on under gart")
}
