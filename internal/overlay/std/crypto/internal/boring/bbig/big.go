//go:build dart

// This file patches the file big.go of the package
// crypto/internal/boring/bbig.
//
// The real file reads the words of a big.Int as a slice of uint through the
// package unsafe, thus the 2 slices share the storage. This file copies the
// words instead. BoringCrypto never stands under gart, thus no call of Enc or
// of Dec ever runs.
package bbig

import (
	"crypto/internal/boring"
	"math/big"
)

// Enc gives the words of b in the form of BoringCrypto.
func Enc(b *big.Int) boring.BigInt {
	if b == nil {
		return nil
	}
	x := b.Bits()
	if len(x) == 0 {
		return boring.BigInt{}
	}
	out := make(boring.BigInt, len(x))
	for i, w := range x {
		out[i] = uint(w)
	}
	return out
}

// Dec gives the number of the words of BoringCrypto.
func Dec(b boring.BigInt) *big.Int {
	if b == nil {
		return nil
	}
	if len(b) == 0 {
		return new(big.Int)
	}
	x := make([]big.Word, len(b))
	for i, w := range b {
		x[i] = big.Word(w)
	}
	return new(big.Int).SetBits(x)
}
