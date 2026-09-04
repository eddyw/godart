// The program proves that HPKE seals, opens, and rejects a changed ciphertext.
package main

import (
	"bytes"
	"crypto/hpke"
	"fmt"
)

func main() {
	kem, kdf, aead := hpke.MLKEM768X25519(), hpke.HKDFSHA256(), hpke.AES256GCM()
	fmt.Println("ids", kem.ID(), kdf.ID(), aead.ID())

	key, err := kem.GenerateKey()
	if err != nil {
		panic(err)
	}
	pubBytes := key.PublicKey().Bytes()
	fmt.Println("public key", len(pubBytes))

	pub, err := kem.NewPublicKey(pubBytes)
	if err != nil {
		panic(err)
	}

	message := []byte("|-()-|")
	info := []byte("example")
	ct, err := hpke.Seal(pub, kdf, aead, info, message)
	if err != nil {
		panic(err)
	}

	plain, err := hpke.Open(key, kdf, aead, info, ct)
	if err != nil {
		panic(err)
	}
	fmt.Printf("opened %s\n", plain)

	bad := bytes.Clone(ct)
	bad[len(bad)-1] ^= 0xff
	if _, err := hpke.Open(key, kdf, aead, info, bad); err == nil {
		panic("a changed ciphertext must give an error")
	}
	fmt.Println("changed ciphertext rejected")

	if _, err := hpke.Open(key, kdf, aead, []byte("wrong"), ct); err == nil {
		panic("a wrong info must give an error")
	}
	fmt.Println("wrong info rejected")

	enc, sender, err := hpke.NewSender(pub, kdf, aead, info)
	if err != nil {
		panic(err)
	}
	first, err := sender.Seal([]byte("aad1"), []byte("one"))
	if err != nil {
		panic(err)
	}
	second, err := sender.Seal([]byte("aad2"), []byte("two"))
	if err != nil {
		panic(err)
	}

	recipient, err := hpke.NewRecipient(enc, key, kdf, aead, info)
	if err != nil {
		panic(err)
	}
	p1, err := recipient.Open([]byte("aad1"), first)
	if err != nil {
		panic(err)
	}
	p2, err := recipient.Open([]byte("aad2"), second)
	if err != nil {
		panic(err)
	}
	fmt.Printf("stream %s %s\n", p1, p2)

	sExport, err := sender.Export("witness", 32)
	if err != nil {
		panic(err)
	}
	rExport, err := recipient.Export("witness", 32)
	if err != nil {
		panic(err)
	}
	fmt.Println("export agrees", bytes.Equal(sExport, rExport), len(sExport))
}
