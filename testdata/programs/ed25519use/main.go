// Command ed25519use reads the signature of crypto/ed25519. Each answer below
// comes from the test vectors of RFC 8032, section 7.1.
package main

import (
	"bytes"
	"crypto"
	"crypto/ed25519"
	"crypto/sha512"
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

func run(seed, msg string) {
	priv := ed25519.NewKeyFromSeed(unhex(seed))
	pub := priv.Public().(ed25519.PublicKey)
	fmt.Println(hex.EncodeToString(pub))
	m := unhex(msg)
	sig := ed25519.Sign(priv, m)
	fmt.Println(hex.EncodeToString(sig))
	fmt.Println(ed25519.Verify(pub, m, sig))
	sig[0] ^= 1
	fmt.Println(ed25519.Verify(pub, m, sig))
	fmt.Println(bytes.Equal(priv.Seed(), unhex(seed)))
}

func main() {
	fmt.Println(ed25519.PublicKeySize, ed25519.PrivateKeySize,
		ed25519.SignatureSize, ed25519.SeedSize)

	// TEST 1 of RFC 8032, with the empty message.
	run("9d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60", "")
	// TEST 2 of RFC 8032, with 1 byte of message.
	run("4ccd089b28ff96da9db6c346ec114e0f5b8a319f35aba624da8cf6ed4fb8a6fb", "72")
	// TEST 3 of RFC 8032, with 2 bytes of message.
	run("c5aa8df43f9f837bedb7442f31dcb7b166d38535076f094b85ce3a2e0b4458f7", "af82")

	// The key from the seed, and the pair of the same seed, agree.
	pub, priv, err := ed25519.GenerateKey(bytes.NewReader(
		unhex("9d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60")))
	must(err)
	fmt.Println(hex.EncodeToString(pub))
	fmt.Println(priv.Equal(ed25519.NewKeyFromSeed(
		unhex("9d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60"))))
	fmt.Println(pub.Equal(priv.Public()))

	// The Ed25519ph form, with a prehash of SHA-512.
	msg := []byte("abc")
	sum := sha512.Sum512(msg)
	p2 := ed25519.NewKeyFromSeed(unhex(
		"833fe62409237b9d62ec77587520911e9a759cec1d19755b7da901b96dca3d42"))
	sig, err := p2.Sign(nil, sum[:], crypto.SHA512)
	must(err)
	fmt.Println(hex.EncodeToString(sig))
	fmt.Println(ed25519.VerifyWithOptions(p2.Public().(ed25519.PublicKey), sum[:], sig,
		&ed25519.Options{Hash: crypto.SHA512}))

	// The Ed25519ctx form, with a context.
	p3 := ed25519.NewKeyFromSeed(unhex(
		"0305334e381af78f141cb666f6199f57bc3495335a256a95bd2a55bf546663f6"))
	sig3, err := p3.Sign(nil, unhex("f726936d19c800494e3fdaff20b276a8"),
		&ed25519.Options{Context: "foo"})
	must(err)
	fmt.Println(hex.EncodeToString(sig3))
	fmt.Println(ed25519.VerifyWithOptions(p3.Public().(ed25519.PublicKey),
		unhex("f726936d19c800494e3fdaff20b276a8"), sig3,
		&ed25519.Options{Context: "foo"}))
}
