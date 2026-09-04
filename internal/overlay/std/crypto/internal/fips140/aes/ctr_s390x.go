//go:build dart

// This file patches the file ctr_s390x.go of the package crypto/internal/fips140/aes.
//
// The real file declares each counter function with no body, because assembly
// holds the code. This file holds the body of ctr_noasm.go, thus each caller
// takes the generic path.
package aes

func ctrBlocks1(b *Block, dst, src *[BlockSize]byte, ivlo, ivhi uint64) {
	ctrBlocks(b, dst[:], src[:], ivlo, ivhi)
}

func ctrBlocks2(b *Block, dst, src *[2 * BlockSize]byte, ivlo, ivhi uint64) {
	ctrBlocks(b, dst[:], src[:], ivlo, ivhi)
}

func ctrBlocks4(b *Block, dst, src *[4 * BlockSize]byte, ivlo, ivhi uint64) {
	ctrBlocks(b, dst[:], src[:], ivlo, ivhi)
}

func ctrBlocks8(b *Block, dst, src *[8 * BlockSize]byte, ivlo, ivhi uint64) {
	ctrBlocks(b, dst[:], src[:], ivlo, ivhi)
}
