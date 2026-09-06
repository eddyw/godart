// The witness of the verb %x on a float32. Go shifts the 23 bits of the
// fraction 1 place to the left, thus the text holds 6 hexadecimal digits.
//
// Go:   0x1.fffffep+127
// gart: 0x1.7fffffp+127
package main

import (
	"fmt"
	"math"
)

func main() {
	fmt.Printf("%x\n", float32(math.MaxFloat32))
	fmt.Printf("%x\n", float32(1.0/3.0))
	fmt.Printf("%x\n", float32(math.Pi))
	fmt.Printf("%x\n", math.MaxFloat64)
}
