// Command ecdhuse reads the key agreement of crypto/ecdh. Each answer below
// comes from a fixed private key and from a published test vector.
package main

import (
	"crypto/ecdh"
	"encoding/hex"
	"fmt"
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

func run(name string, c ecdh.Curve, a, b string) {
	ka, err := c.NewPrivateKey(unhex(a))
	must(err)
	kb, err := c.NewPrivateKey(unhex(b))
	must(err)
	fmt.Println(name, ka.Curve() == c)
	fmt.Println(hex.EncodeToString(ka.PublicKey().Bytes()))
	fmt.Println(hex.EncodeToString(kb.PublicKey().Bytes()))
	s1, err := ka.ECDH(kb.PublicKey())
	must(err)
	s2, err := kb.ECDH(ka.PublicKey())
	must(err)
	fmt.Println(hex.EncodeToString(s1))
	fmt.Println(hex.EncodeToString(s2))
	fmt.Println(ka.PublicKey().Equal(ka.PublicKey()), ka.Equal(kb))

	// The read of the public bytes again gives the same key.
	pb, err := c.NewPublicKey(ka.PublicKey().Bytes())
	must(err)
	fmt.Println(pb.Equal(ka.PublicKey()))
	fmt.Println()
}

func main() {
	// RFC 7748, section 6.1: the 2 private keys of the X25519 example.
	run("X25519", ecdh.X25519(),
		"77076d0a7318a57d3c16c17251b26645df4c2f87ebc0992ab177fba51db92c2a",
		"5dab087e624a8a4b79e17f8b83800ee66f3bb1292618b6fd1c2f8b27ff88e0eb")

	// A fixed scalar for each NIST curve of the package.
	run("P256", ecdh.P256(),
		"0101010101010101010101010101010101010101010101010101010101010101",
		"0202020202020202020202020202020202020202020202020202020202020202")
	run("P384", ecdh.P384(),
		"010101010101010101010101010101010101010101010101"+
			"010101010101010101010101010101010101010101010101",
		"020202020202020202020202020202020202020202020202"+
			"020202020202020202020202020202020202020202020202")
	run("P521", ecdh.P521(),
		"0001010101010101010101010101010101010101010101010101010101010101"+
			"0101010101010101010101010101010101010101010101010101010101010101"+
			"0101",
		"0002020202020202020202020202020202020202020202020202020202020202"+
			"0202020202020202020202020202020202020202020202020202020202020202"+
			"0202")

	// A key of the wrong size gives an error.
	_, err := ecdh.X25519().NewPrivateKey(make([]byte, 31))
	fmt.Println(err)
	_, err = ecdh.P256().NewPublicKey(make([]byte, 10))
	fmt.Println(err)
}
