//go:build dart

// This file patches the file nat_wasm.go of the package
// crypto/internal/fips140/bigmod. The real file holds the 3 special forms of
// addMulVVW, and the patch of nat.go names the generic form instead.
package bigmod
