//go:build dart

// This file patches the file chacha_arm64.go of the package
// vendor/golang.org/x/crypto/chacha20.
//
// The real file declares xorKeyStreamVX with no body, and an assembly file of
// arm64 holds the code. The body below is the body of chacha_noasm.go, which
// Go builds for each machine that holds no such file.
package chacha20

const bufSize = blockSize

func (s *Cipher) xorKeyStreamBlocks(dst, src []byte) {
	s.xorKeyStreamBlocksGeneric(dst, src)
}
