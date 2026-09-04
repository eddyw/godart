// Command sha3use reads the SHA-3 family and the SHAKE family. Each answer
// below comes from the test vectors of NIST for FIPS 202.
package main

import (
	"crypto"
	"crypto/hmac"
	"crypto/sha3"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
)

func main() {
	empty := []byte("")
	abc := []byte("abc")

	s224 := sha3.Sum224(empty)
	fmt.Println(hex.EncodeToString(s224[:]))
	s256 := sha3.Sum256(empty)
	fmt.Println(hex.EncodeToString(s256[:]))
	s384 := sha3.Sum384(empty)
	fmt.Println(hex.EncodeToString(s384[:]))
	s512 := sha3.Sum512(empty)
	fmt.Println(hex.EncodeToString(s512[:]))

	a224 := sha3.Sum224(abc)
	fmt.Println(hex.EncodeToString(a224[:]))
	a256 := sha3.Sum256(abc)
	fmt.Println(hex.EncodeToString(a256[:]))
	a384 := sha3.Sum384(abc)
	fmt.Println(hex.EncodeToString(a384[:]))
	a512 := sha3.Sum512(abc)
	fmt.Println(hex.EncodeToString(a512[:]))

	// The streaming form gives the same answer.
	h := sha3.New256()
	h.Write([]byte("a"))
	h.Write([]byte("bc"))
	fmt.Println(hex.EncodeToString(h.Sum(nil)))
	fmt.Println(h.Size(), h.BlockSize())
	h.Reset()
	fmt.Println(hex.EncodeToString(h.Sum(nil)))

	// SHAKE128 and SHAKE256 over the empty input, 32 bytes of each.
	sh128 := sha3.NewSHAKE128()
	out := make([]byte, 32)
	io.ReadFull(sh128, out)
	fmt.Println(hex.EncodeToString(out))
	sh256 := sha3.NewSHAKE256()
	out2 := make([]byte, 32)
	io.ReadFull(sh256, out2)
	fmt.Println(hex.EncodeToString(out2))

	// SumSHAKE128 and SumSHAKE256 of the same input.
	fmt.Println(hex.EncodeToString(sha3.SumSHAKE128(empty, 32)))
	fmt.Println(hex.EncodeToString(sha3.SumSHAKE256(empty, 32)))

	// cSHAKE128 with the function name and the customization of the sample of
	// NIST.
	cs := sha3.NewCSHAKE128([]byte(""), []byte("Email Signature"))
	cs.Write([]byte{0x00, 0x01, 0x02, 0x03})
	out3 := make([]byte, 32)
	io.ReadFull(cs, out3)
	fmt.Println(hex.EncodeToString(out3))

	// The hash of the package crypto reaches SHA3 after the import.
	fmt.Println(crypto.SHA3_256.Available(), crypto.SHA3_512.Available())
	n := crypto.SHA3_256.New()
	n.Write(abc)
	fmt.Println(hex.EncodeToString(n.Sum(nil)))

	// HMAC over SHA3-256, with the key of 32 bytes of 0x0b.
	key := make([]byte, 32)
	for i := range key {
		key[i] = 0x0b
	}
	m := hmac.New(func() hash.Hash { return sha3.New256() }, key)
	m.Write([]byte("Hi There"))
	fmt.Println(hex.EncodeToString(m.Sum(nil)))

	// The marshal of the state and the read of it again.
	h2 := sha3.New256()
	h2.Write([]byte("a"))
	state, err := h2.MarshalBinary()
	if err != nil {
		panic(err)
	}
	h3 := sha3.New256()
	if err := h3.UnmarshalBinary(state); err != nil {
		panic(err)
	}
	h3.Write([]byte("bc"))
	fmt.Println(hex.EncodeToString(h3.Sum(nil)))
}
