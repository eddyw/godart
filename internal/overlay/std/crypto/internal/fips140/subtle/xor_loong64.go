//go:build dart

// This file patches the file xor_loong64.go of the package
// crypto/internal/fips140/subtle. The real file declares or holds xorBytes,
// and the patch of xor.go holds the loop itself.
package subtle
