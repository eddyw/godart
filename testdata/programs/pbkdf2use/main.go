// Command pbkdf2use reads the key derivation of RFC 2898. Each answer below
// comes from the test vectors of RFC 6070 and of RFC 7914.
package main

import (
	"crypto/pbkdf2"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	// RFC 6070, the 4 first vectors with SHA-1.
	k1, err := pbkdf2.Key(sha1.New, "password", []byte("salt"), 1, 20)
	must(err)
	fmt.Println(hex.EncodeToString(k1))
	k2, err := pbkdf2.Key(sha1.New, "password", []byte("salt"), 2, 20)
	must(err)
	fmt.Println(hex.EncodeToString(k2))
	k3, err := pbkdf2.Key(sha1.New, "password", []byte("salt"), 4096, 20)
	must(err)
	fmt.Println(hex.EncodeToString(k3))
	k4, err := pbkdf2.Key(sha1.New, "passwordPASSWORDpassword",
		[]byte("saltSALTsaltSALTsaltSALTsaltSALTsalt"), 4096, 25)
	must(err)
	fmt.Println(hex.EncodeToString(k4))

	// RFC 7914, section 11: the 2 vectors with SHA-256.
	k5, err := pbkdf2.Key(sha256.New, "passwd", []byte("salt"), 1, 64)
	must(err)
	fmt.Println(hex.EncodeToString(k5))
	k6, err := pbkdf2.Key(sha256.New, "Password", []byte("NaCl"), 80000, 64)
	must(err)
	fmt.Println(hex.EncodeToString(k6))

	_, err = pbkdf2.Key(sha256.New, "x", []byte("y"), 1, 0)
	fmt.Println(err)
}
