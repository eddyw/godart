// Command hkdfuse reads the key derivation of RFC 5869. Each answer below comes
// from the appendix A of that document.
package main

import (
	"crypto/hkdf"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func unhex(s string) []byte {
	b, err := hex.DecodeString(s)
	must(err)
	return b
}

func main() {
	// Test case 1 of RFC 5869, with SHA-256.
	ikm := unhex("0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b")
	salt := unhex("000102030405060708090a0b0c")
	info := unhex("f0f1f2f3f4f5f6f7f8f9")
	prk, err := hkdf.Extract(sha256.New, ikm, salt)
	must(err)
	fmt.Println(hex.EncodeToString(prk))
	okm, err := hkdf.Expand(sha256.New, prk, string(info), 42)
	must(err)
	fmt.Println(hex.EncodeToString(okm))
	key, err := hkdf.Key(sha256.New, ikm, salt, string(info), 42)
	must(err)
	fmt.Println(hex.EncodeToString(key))

	// Test case 3 of RFC 5869: no salt and no info, with SHA-256.
	prk3, err := hkdf.Extract(sha256.New, ikm, nil)
	must(err)
	fmt.Println(hex.EncodeToString(prk3))
	okm3, err := hkdf.Expand(sha256.New, prk3, "", 42)
	must(err)
	fmt.Println(hex.EncodeToString(okm3))

	// Test case 4 of RFC 5869, with SHA-1. The key of the input holds 11 bytes.
	ikm4 := unhex("0b0b0b0b0b0b0b0b0b0b0b")
	prk4, err := hkdf.Extract(func() hash.Hash { return sha1.New() }, ikm4, salt)
	must(err)
	fmt.Println(hex.EncodeToString(prk4))
	okm4, err := hkdf.Expand(sha1.New, prk4, string(info), 42)
	must(err)
	fmt.Println(hex.EncodeToString(okm4))

	// A length above the limit gives an error.
	_, err = hkdf.Expand(sha256.New, prk, "", 255*32+1)
	fmt.Println(err)
}
