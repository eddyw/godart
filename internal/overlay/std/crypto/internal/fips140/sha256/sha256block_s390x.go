//go:build dart

// This file patches the file sha256block_s390x.go of the package
// crypto/internal/fips140/sha256.
//
// The real file declares the block function with no body, because assembly
// holds the code. This file names the generic form instead.
package sha256

func block(dig *Digest, p []byte) {
	blockGeneric(dig, p)
}
