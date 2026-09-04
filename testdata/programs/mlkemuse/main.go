// The program proves that ML-KEM encapsulates and decapsulates 1 shared key.
package main

import (
	"bytes"
	"crypto/mlkem"
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

	if _, err := mlkem.NewEncapsulationKey768([]byte{1, 2, 3}); err == nil {
		panic("a short encapsulation key must give an error")
	}
	fmt.Println("short key rejected")
}
