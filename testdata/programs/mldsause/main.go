// The program proves that ML-DSA signs, verifies, and rejects a forged
// signature.
package main

import (
	"bytes"
	"crypto/mldsa"
	"fmt"
)

func main() {
	for _, p := range []mldsa.Parameters{mldsa.MLDSA44(), mldsa.MLDSA65(), mldsa.MLDSA87()} {
		fmt.Println(p.String(), p.PublicKeySize(), p.SignatureSize())
	}

	params := mldsa.MLDSA44()
	sk, err := mldsa.GenerateKey(params)
	if err != nil {
		panic(err)
	}
	pubBytes := sk.PublicKey().Bytes()
	fmt.Println("public key", len(pubBytes))

	msg := []byte("hello, world")
	opts := &mldsa.Options{Context: "witness"}
	sig, err := sk.Sign(nil, msg, opts)
	if err != nil {
		panic(err)
	}
	fmt.Println("signature", len(sig))

	pk, err := mldsa.NewPublicKey(params, pubBytes)
	if err != nil {
		panic(err)
	}
	fmt.Println("verify", mldsa.Verify(pk, msg, sig, opts) == nil)

	forged := bytes.Clone(sig)
	forged[0] ^= 0xff
	fmt.Println("forged rejected", mldsa.Verify(pk, msg, forged, opts) != nil)

	fmt.Println("other message rejected",
		mldsa.Verify(pk, []byte("hello, worle"), sig, opts) != nil)

	fmt.Println("other context rejected",
		mldsa.Verify(pk, msg, sig, &mldsa.Options{Context: "other"}) != nil)

	det1, err := sk.SignDeterministic(msg, opts)
	if err != nil {
		panic(err)
	}
	det2, err := sk.SignDeterministic(msg, opts)
	if err != nil {
		panic(err)
	}
	fmt.Println("deterministic", bytes.Equal(det1, det2))
	fmt.Println("deterministic verify", mldsa.Verify(pk, msg, det1, opts) == nil)

	again, err := mldsa.NewPrivateKey(params, sk.Bytes())
	if err != nil {
		panic(err)
	}
	fmt.Println("seed equal", again.Equal(sk))
	fmt.Println("public equal", again.PublicKey().Equal(pk))
	fmt.Println("parameters", again.PublicKey().Parameters().String())
}
