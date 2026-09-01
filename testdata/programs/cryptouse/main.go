package main

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
)

var vectors = []string{
	"",
	"a",
	"abc",
	"message digest",
	"abcdefghijklmnopqrstuvwxyz",
	"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789",
	"12345678901234567890123456789012345678901234567890123456789012345678901234567890",
}

func main() {
	for _, v := range vectors {
		sum := md5.Sum([]byte(v))
		s256 := sha256.Sum256([]byte(v))
		s224 := sha256.Sum224([]byte(v))
		fmt.Println(hex.EncodeToString(sum[:]))
		fmt.Println(hex.EncodeToString(s256[:]))
		fmt.Println(hex.EncodeToString(s224[:]))
		s1 := sha1.Sum([]byte(v))
		fmt.Println(hex.EncodeToString(s1[:]))
		s512 := sha512.Sum512([]byte(v))
		s384 := sha512.Sum384([]byte(v))
		s512_224 := sha512.Sum512_224([]byte(v))
		s512_256 := sha512.Sum512_256([]byte(v))
		fmt.Println(hex.EncodeToString(s512[:]))
		fmt.Println(hex.EncodeToString(s384[:]))
		fmt.Println(hex.EncodeToString(s512_224[:]))
		fmt.Println(hex.EncodeToString(s512_256[:]))
		fmt.Println(hex.EncodeToString(hmac.New(sha256.New, []byte("a key")).Sum(nil)))
	}

	writeParts(sha1.New())
	writeParts(md5.New())
	writeParts(sha256.New())
	writeParts(sha256.New224())
	writeParts(sha512.New())
	writeParts(sha512.New384())
	writeParts(sha512.New512_224())
	writeParts(sha512.New512_256())
	writeParts(hmac.New(sha1.New, []byte("k")))
	writeParts(hmac.New(sha512.New, []byte("a key that is longer than one block of the hash function, thus the key gives its own hash first")))

	long := make([]byte, 0, 4000)
	for i := 0; i < 4000; i++ {
		long = append(long, byte(i%251))
	}
	h := sha256.New()
	h.Write(long)
	fmt.Println("long", hex.EncodeToString(h.Sum(nil)))

	marshal(sha1.New())
	marshal(md5.New())
	marshal(sha256.New())
	marshal(sha512.New())

	fmt.Println(hmac.Equal([]byte("abc"), []byte("abc")),
		hmac.Equal([]byte("abc"), []byte("abd")),
		hmac.Equal([]byte("ab"), []byte("abc")))

	// The bytes differ at each run, thus the test reads the shape only.
	secret := make([]byte, 32)
	n, err := rand.Read(secret)
	fmt.Println("rand", n, err == nil, len(rand.Text()), allZero(secret))
	other := make([]byte, 32)
	rand.Read(other)
	fmt.Println("rand differs", !sameBytes(secret, other))
	fromReader := make([]byte, 16)
	n, err = rand.Reader.Read(fromReader)
	fmt.Println("reader", n, err == nil)
}

func allZero(b []byte) bool {
	for _, c := range b {
		if c != 0 {
			return false
		}
	}
	return true
}

func sameBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func writeParts(h hash.Hash) {
	h.Write([]byte("the quick "))
	h.Write([]byte("brown fox "))
	h.Write([]byte("jumps over the lazy dog"))
	fmt.Println(hex.EncodeToString(h.Sum(nil)), h.Size(), h.BlockSize())
	h.Reset()
	fmt.Println("after reset", hex.EncodeToString(h.Sum(nil)))
}

func marshal(h hash.Hash) {
	h.Write([]byte("a state that travels"))
	m, ok := h.(interface{ MarshalBinary() ([]byte, error) })
	if !ok {
		fmt.Println("no marshaler")
		return
	}
	state, err := m.MarshalBinary()
	if err != nil {
		fmt.Println("marshal", err)
		return
	}
	fmt.Println("state", len(state))
	h.Write([]byte(" and grows"))
	want := hex.EncodeToString(h.Sum(nil))

	u, ok := h.(interface{ UnmarshalBinary([]byte) error })
	if !ok {
		fmt.Println("no unmarshaler")
		return
	}
	if err := u.UnmarshalBinary(state); err != nil {
		fmt.Println("unmarshal", err)
		return
	}
	h.Write([]byte(" and grows"))
	fmt.Println("round trip", want == hex.EncodeToString(h.Sum(nil)))
}
