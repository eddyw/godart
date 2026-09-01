//go:build dart

// Package rand gives random bytes that a program can use for a secret.
//
// The bytes come from Random.secure of Dart. That source reads the generator of
// the operating system, and the web build reads window.crypto.
//
// Go also holds Int and Prime. The 2 read the package math/big, and that
// package stands outside the surface. docs/parity.md holds that gap.
package rand

import (
	"io"
	"syscall/dart"
)

// Reader is a global source of random bytes.
var Reader io.Reader = reader{}

type reader struct{}

func (reader) Read(b []byte) (int, error) {
	copy(b, dart.RandomBytes(len(b)))
	return len(b), nil
}

// Read fills b with random bytes. It always fills b, and it never gives an
// error.
func Read(b []byte) (int, error) {
	return io.ReadFull(Reader, b)
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
