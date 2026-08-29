package main

import (
	"crypto/md5"
	"crypto/sha256"
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
	}

	writeParts(md5.New())
	writeParts(sha256.New())
	writeParts(sha256.New224())

	long := make([]byte, 0, 4000)
	for i := 0; i < 4000; i++ {
		long = append(long, byte(i%251))
	}
	h := sha256.New()
	h.Write(long)
	fmt.Println("long", hex.EncodeToString(h.Sum(nil)))

	marshal(md5.New())
	marshal(sha256.New())
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
