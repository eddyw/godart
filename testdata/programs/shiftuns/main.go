// The program proves the shift by a count of an unsigned type. Go holds no
// negative count of such a type. A conversion of a negative value gives a
// count above the width, thus the shift gives 0, or -1 for a right shift of a
// negative signed value.
package main

import "fmt"

func main() {
	var ch rune = -1
	var ws uint64 = 1<<'\t' | 1<<'\n' | 1<<' '
	fmt.Println("mask", ws&(1<<uint(ch)))

	var n int8 = -3
	fmt.Println("left", uint64(1)<<uint(n))

	var big int = 200
	fmt.Println("wide", uint32(7)<<uint(big))

	fmt.Println("signed", int64(-8)>>uint(ch))
	fmt.Println("logical", uint64(1<<63)>>uint(ch))

	// A count that fits keeps the plain form.
	var k uint = 3
	fmt.Println("plain", uint64(1)<<k, int64(-64)>>k)
}
