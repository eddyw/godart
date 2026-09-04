//go:build dart

// This file patches the file cbc_ppc64x.go of the package crypto/internal/fips140/aes.
//
// The real file reaches an instruction of the machine. This file holds the body
// of cbc_noasm.go, thus each caller takes the generic path.
package aes

func cryptBlocksEnc(b *Block, civ *[BlockSize]byte, dst, src []byte) {
	cryptBlocksEncGeneric(b, civ, dst, src)
}

func cryptBlocksDec(b *Block, civ *[BlockSize]byte, dst, src []byte) {
	cryptBlocksDecGeneric(b, civ, dst, src)
}
