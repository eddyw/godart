// Command rc4use reads the RC4 stream cipher. Each answer below comes from the
// test vectors of RFC 6229 and of the appendix A of the draft of Kaukonen.
package main

import (
	"crypto/rc4"
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
	// RFC 6229, the key 0102030405 over 16 bytes of 0.
	c, err := rc4.NewCipher(unhex("0102030405"))
	must(err)
	out := make([]byte, 16)
	c.XORKeyStream(out, make([]byte, 16))
	fmt.Println(hex.EncodeToString(out))

	// The vector "Key" over "Plaintext" of the appendix of the draft.
	c2, err := rc4.NewCipher([]byte("Key"))
	must(err)
	out2 := make([]byte, len("Plaintext"))
	c2.XORKeyStream(out2, []byte("Plaintext"))
	fmt.Println(hex.EncodeToString(out2))

	// The vector "Wiki" over "pedia".
	c3, err := rc4.NewCipher([]byte("Wiki"))
	must(err)
	out3 := make([]byte, len("pedia"))
	c3.XORKeyStream(out3, []byte("pedia"))
	fmt.Println(hex.EncodeToString(out3))

	// The vector "Secret" over "Attack at dawn".
	c4, err := rc4.NewCipher([]byte("Secret"))
	must(err)
	out4 := make([]byte, len("Attack at dawn"))
	c4.XORKeyStream(out4, []byte("Attack at dawn"))
	fmt.Println(hex.EncodeToString(out4))

	// The same cipher over the output gives the input again.
	c5, err := rc4.NewCipher([]byte("Secret"))
	must(err)
	back := make([]byte, len(out4))
	c5.XORKeyStream(back, out4)
	fmt.Println(string(back))

	c6, err := rc4.NewCipher(unhex("0102030405"))
	must(err)
	c6.Reset()
	fmt.Println("reset")

	_, err = rc4.NewCipher(make([]byte, 0))
	fmt.Println(err)
	var e rc4.KeySizeError = 300
	fmt.Println(e.Error())
}
