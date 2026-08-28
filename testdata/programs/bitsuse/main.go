package main

import (
	"fmt"
	"math/bits"
)

func main() {
	fmt.Println(bits.OnesCount(255), bits.OnesCount64(1<<40))
	fmt.Println(bits.LeadingZeros8(1), bits.TrailingZeros16(8))
	fmt.Println(bits.Len(255), bits.Len64(1<<40))
	fmt.Println(bits.Reverse8(1), bits.ReverseBytes32(1))
	fmt.Println(bits.RotateLeft8(1, 1), bits.UintSize)
	hi, lo := bits.Mul64(1<<40, 1<<40)
	fmt.Println(hi, lo)
	q, r := bits.Div64(0, 100, 7)
	fmt.Println(q, r)
	s, c := bits.Add64(1<<63, 1<<63, 0)
	fmt.Println(s, c)
	d, b := bits.Sub64(1, 2, 0)
	fmt.Println(d, b)
	fmt.Println(bits.OnesCount32(0xFFFF), bits.TrailingZeros(0))
}
