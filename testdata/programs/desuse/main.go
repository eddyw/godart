// Command desuse reads the DES block cipher and the triple DES block cipher.
// Each answer below comes from a published test vector.
package main

import (
	"crypto/cipher"
	"crypto/des"
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

func main() {
	fmt.Println(des.BlockSize)

	// The known answer test of NIST SP 800-20, and the vector of the book of
	// Schneier: the key 0123456789abcdef over the text 4e6f772069732074.
	block, err := des.NewCipher(unhex("0123456789abcdef"))
	must(err)
	fmt.Println(block.BlockSize())
	out := make([]byte, des.BlockSize)
	block.Encrypt(out, unhex("4e6f772069732074"))
	fmt.Println(hex.EncodeToString(out))
	back := make([]byte, des.BlockSize)
	block.Decrypt(back, out)
	fmt.Println(hex.EncodeToString(back))

	// The all zero key over the all zero block, of the known answer set.
	zero, err := des.NewCipher(make([]byte, 8))
	must(err)
	zout := make([]byte, des.BlockSize)
	zero.Encrypt(zout, make([]byte, 8))
	fmt.Println(hex.EncodeToString(zout))

	// Triple DES with 3 equal keys gives the answer of 1 DES.
	k := unhex("0123456789abcdef")
	tri, err := des.NewTripleDESCipher(append(append(append([]byte{}, k...), k...), k...))
	must(err)
	fmt.Println(tri.BlockSize())
	tout := make([]byte, des.BlockSize)
	tri.Encrypt(tout, unhex("4e6f772069732074"))
	fmt.Println(hex.EncodeToString(tout))

	// Triple DES with 3 different keys, from the vector set of NIST.
	tri2, err := des.NewTripleDESCipher(unhex(
		"0123456789abcdef23456789abcdef01456789abcdef0123"))
	must(err)
	t2 := make([]byte, des.BlockSize)
	tri2.Encrypt(t2, unhex("54686520717566636b2062726f776e20"[:16]))
	fmt.Println(hex.EncodeToString(t2))
	b2 := make([]byte, des.BlockSize)
	tri2.Decrypt(b2, t2)
	fmt.Println(hex.EncodeToString(b2))

	// CBC over the DES block, with the mode of crypto/cipher.
	iv := unhex("1234567890abcdef")
	text := unhex("4e6f77206973207468652074696d6520")
	ct := make([]byte, len(text))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ct, text)
	fmt.Println(hex.EncodeToString(ct))
	pt := make([]byte, len(ct))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(pt, ct)
	fmt.Println(hex.EncodeToString(pt))

	_, err = des.NewCipher(make([]byte, 7))
	fmt.Println(err)
	_, err = des.NewTripleDESCipher(make([]byte, 16))
	fmt.Println(err)
	var e des.KeySizeError = 5
	fmt.Println(e.Error())
}
