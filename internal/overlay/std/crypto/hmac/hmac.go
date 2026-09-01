//go:build dart

// Package hmac implements the Keyed-Hash Message Authentication Code of
// FIPS 198-1.
//
// The overlay holds the algorithm of crypto/internal/fips140/hmac, with the
// public names of crypto/hmac. It drops the registration of the package crypto,
// the boring path, and the FIPS mode of Go.
package hmac

import "hash"

// New gives a hash.Hash that writes the HMAC of the key with the hash function
// h. Each call of h must give a new hash of the same kind.
func New(h func() hash.Hash, key []byte) hash.Hash {
	m := &hmac{outer: h(), inner: h()}
	m.size = m.inner.Size()
	m.blockSize = m.inner.BlockSize()
	m.ipad = make([]byte, m.blockSize)
	m.opad = make([]byte, m.blockSize)
	if len(key) > m.blockSize {
		// A key that is longer than 1 block gives its own hash.
		m.outer.Write(key)
		key = m.outer.Sum(nil)
	}
	copy(m.ipad, key)
	copy(m.opad, key)
	for i := range m.ipad {
		m.ipad[i] ^= 0x36
	}
	for i := range m.opad {
		m.opad[i] ^= 0x5c
	}
	m.inner.Write(m.ipad)
	return m
}

// Equal compares 2 codes. It takes the same time for each pair of the same
// length, thus a caller gives away no byte of the right code.
func Equal(mac1, mac2 []byte) bool {
	if len(mac1) != len(mac2) {
		return false
	}
	var out byte
	for i := range mac1 {
		out |= mac1[i] ^ mac2[i]
	}
	return out == 0
}

type hmac struct {
	opad      []byte
	ipad      []byte
	outer     hash.Hash
	inner     hash.Hash
	size      int
	blockSize int
}

func (h *hmac) Write(p []byte) (int, error) { return h.inner.Write(p) }

func (h *hmac) Size() int { return h.size }

func (h *hmac) BlockSize() int { return h.blockSize }

func (h *hmac) Reset() {
	h.inner.Reset()
	h.inner.Write(h.ipad)
}

func (h *hmac) Sum(in []byte) []byte {
	sum := h.inner.Sum(nil)
	h.outer.Reset()
	h.outer.Write(h.opad)
	h.outer.Write(sum)
	return h.outer.Sum(in)
}
