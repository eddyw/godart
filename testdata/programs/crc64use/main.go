package main

import (
	"fmt"
	"hash/crc64"
)

func main() {
	iso := crc64.MakeTable(crc64.ISO)
	ecma := crc64.MakeTable(crc64.ECMA)
	fmt.Printf("iso:  %016x\n", crc64.Checksum([]byte("the quick brown fox"), iso))
	fmt.Printf("ecma: %016x\n", crc64.Checksum([]byte("the quick brown fox"), ecma))

	h := crc64.New(ecma)
	h.Write([]byte("the quick "))
	h.Write([]byte("brown fox"))
	fmt.Printf("sum:  %016x\n", h.Sum64())
	fmt.Println("size:", h.Size(), "block:", h.BlockSize())
	fmt.Printf("bytes: %x\n", h.Sum(nil))

	state, err := h.(interface{ MarshalBinary() ([]byte, error) }).MarshalBinary()
	fmt.Println("marshal ok:", err == nil, len(state) > 0)
	g := crc64.New(ecma)
	err = g.(interface{ UnmarshalBinary([]byte) error }).UnmarshalBinary(state)
	fmt.Println("unmarshal ok:", err == nil)
	fmt.Printf("again: %016x\n", g.(interface{ Sum64() uint64 }).Sum64())

	fmt.Printf("update: %016x\n", crc64.Update(0, iso, []byte("ab")))
}
