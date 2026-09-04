//go:build dart

// This file patches the file sha512block_asm.go of the package
// crypto/internal/fips140/sha512.
//
// The real file declares the block function with no body, because assembly
// holds the code. This file names the generic form instead.
package sha512

func block(dig *Digest, p []byte) {
	blockGeneric(dig, p)
}
