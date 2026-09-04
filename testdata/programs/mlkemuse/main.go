// The program proves that ML-KEM encapsulates and decapsulates 1 shared key.
package main

import (
	"bytes"
	"crypto/mlkem"
	"crypto/mlkem/mlkemtest"
	"fmt"
)

func main() {
	dk, err := mlkem.GenerateKey768()
	if err != nil {
		panic(err)
	}
	ekBytes := dk.EncapsulationKey().Bytes()
	fmt.Println("encapsulation key", len(ekBytes))

	ek, err := mlkem.NewEncapsulationKey768(ekBytes)
	if err != nil {
		panic(err)
	}
	shared, ct := ek.Encapsulate()
	fmt.Println("shared", len(shared), "ciphertext", len(ct))

	back, err := dk.Decapsulate(ct)
	if err != nil {
		panic(err)
	}
	fmt.Println("agree", bytes.Equal(shared, back))

	bad := bytes.Clone(ct)
	bad[0] ^= 0xff
	other, err := dk.Decapsulate(bad)
	if err != nil {
		panic(err)
	}
	fmt.Println("forged agrees", bytes.Equal(shared, other))

	seed := dk.Bytes()
	again, err := mlkem.NewDecapsulationKey768(seed)
	if err != nil {
		panic(err)
	}
	repeat, err := again.Decapsulate(ct)
	if err != nil {
		panic(err)
	}
	fmt.Println("seed agrees", bytes.Equal(shared, repeat))

	dk1024, err := mlkem.GenerateKey1024()
	if err != nil {
		panic(err)
	}
	s1, c1 := dk1024.EncapsulationKey().Encapsulate()
	r1, err := dk1024.Decapsulate(c1)
	if err != nil {
		panic(err)
	}
	fmt.Println("1024 agree", bytes.Equal(s1, r1), len(s1))

	seedBytes := make([]byte, 32)
	for i := range seedBytes {
		seedBytes[i] = byte(i)
	}
	fixed1, fixedCT, err := mlkemtest.Encapsulate768(ek, seedBytes)
	if err != nil {
		panic(err)
	}
	fixed2, _, err := mlkemtest.Encapsulate768(ek, seedBytes)
	if err != nil {
		panic(err)
	}
	fmt.Println("fixed random agrees", bytes.Equal(fixed1, fixed2))
	opened, err := dk.Decapsulate(fixedCT)
	if err != nil {
		panic(err)
	}
	fmt.Println("fixed decapsulates", bytes.Equal(fixed1, opened))

	if _, _, err := mlkemtest.Encapsulate768(ek, []byte{1}); err == nil {
		panic("a short random must give an error")
	}
	fmt.Println("short random rejected")

	if _, err := mlkem.NewEncapsulationKey768([]byte{1, 2, 3}); err == nil {
		panic("a short encapsulation key must give an error")
	}
	fmt.Println("short key rejected")
}
