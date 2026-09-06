// The witness of a conversion of a float64 to a uint64 above 2 to the power
// 63. The value 2 to the power 63 stands inside the range of a uint64, thus
// the result is exact in Go.
//
// Go:   9223372036854775808
// gart: 9223372036854775807
package main

import "fmt"

func main() {
	f := float64(1 << 63)
	fmt.Println(uint64(f))
	// 1.8e19 stands under 2 to the power 64, thus the result is exact. A
	// value at or above that bound is implementation defined, and gc itself
	// differs between arm64 and amd64.
	g := 1.8e19
	fmt.Println(uint64(g), uint64(g) > 1<<63)
}
