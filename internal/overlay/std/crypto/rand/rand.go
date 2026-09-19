//go:build dart

// Package rand gives random bytes that a program can use for a secret.
//
// The bytes come from Random.secure of Dart. That source reads the generator of
// the operating system, and the web build reads window.crypto.
package rand

import (
	internalrand "crypto/internal/rand"
	"errors"
	"io"
	"math/big"
	"syscall/dart"
)

// Reader is a global source of random bytes.
var Reader io.Reader = internalrand.Reader

// Read fills b with random bytes. It always fills b, and it never gives an
// error.
func Read(b []byte) (int, error) {
	if _, err := io.ReadFull(Reader, b); err != nil {
		dart.Stderr("fatal error: crypto/rand: failed to read random data (see https://go.dev/issue/66821): " + err.Error() + "\n")
		dart.Exit(2)
	}
	return len(b), nil
}

const base32alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"

// Text gives a random text of 26 characters of the base32 alphabet of RFC 4648.
// The text holds 128 bits of randomness, thus a program can use it for a secret
// or for a token.
func Text() string {
	src := make([]byte, 26)
	Read(src)
	for i := range src {
		src[i] = base32alphabet[src[i]%32]
	}
	return string(src)
}

// Prime gives a number of the named bit count that is prime with a high
// probability.
//
// Prime uses the shared secure source unless the Go compatibility setting
// selects the supplied reader.
func Prime(r io.Reader, bits int) (*big.Int, error) {
	if bits < 2 {
		return nil, errors.New("crypto/rand: prime size must be at least 2-bit")
	}
	r = internalrand.CustomReader(r)

	b := uint(bits % 8)
	if b == 0 {
		b = 8
	}

	bytes := make([]byte, (bits+7)/8)
	p := new(big.Int)

	for {
		if _, err := io.ReadFull(r, bytes); err != nil {
			return nil, err
		}

		// The clear of the first byte holds the candidate at the named size,
		// and the set of the top 2 bits holds it above half of that size. The
		// product of 2 such values thus never falls 1 bit short.
		bytes[0] &= uint8(int(1<<b) - 1)
		if b >= 2 {
			bytes[0] |= 3 << (b - 2)
		} else {
			bytes[0] |= 1
			if len(bytes) > 1 {
				bytes[1] |= 0x80
			}
		}
		// An even number of this size is never prime.
		bytes[len(bytes)-1] |= 1

		p.SetBytes(bytes)
		if p.ProbablyPrime(20) {
			return p, nil
		}
	}
}

// Int gives a uniform random value in the range 0 to max, and it holds max
// outside that range. It panics when max is 0 or less.
func Int(rand io.Reader, max *big.Int) (n *big.Int, err error) {
	if max.Sign() <= 0 {
		panic("crypto/rand: argument to Int is <= 0")
	}
	n = new(big.Int)
	n.Sub(max, n.SetUint64(1))
	bitLen := n.BitLen()
	if bitLen == 0 {
		return
	}
	k := (bitLen + 7) / 8
	b := uint(bitLen % 8)
	if b == 0 {
		b = 8
	}

	bytes := make([]byte, k)

	for {
		_, err = io.ReadFull(rand, bytes)
		if err != nil {
			return nil, err
		}

		// The clear of the first byte raises the probability that the
		// candidate falls below max, thus the loop repeats less often.
		bytes[0] &= uint8(int(1<<b) - 1)

		n.SetBytes(bytes)
		if n.Cmp(max) < 0 {
			return
		}
	}
}
