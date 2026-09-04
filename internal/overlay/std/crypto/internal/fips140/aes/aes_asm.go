//go:build dart

// This file patches the file aes_asm.go of the package crypto/internal/fips140/aes.
//
// The real file declares each block function with no body, because assembly
// holds the code. This file holds the body of aes_noasm.go, thus each caller
// takes the generic path.
package aes

type block struct {
	blockExpanded
}

func newBlock(c *Block, key []byte) *Block {
	newBlockExpanded(&c.blockExpanded, key)
	return c
}

func encryptBlock(c *Block, dst, src []byte) {
	encryptBlockGeneric(&c.blockExpanded, dst, src)
}

func decryptBlock(c *Block, dst, src []byte) {
	decryptBlockGeneric(&c.blockExpanded, dst, src)
}

func checkGenericIsExpected() {}
