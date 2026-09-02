package main

import (
	"fmt"
	"math/big"
	"strings"
)

// The overlay patches ftoa.go, intconv.go, and natconv.go of math/big. This
// program reads each line that the 3 patches touch.
func main() {
	i := big.NewInt(-1234)
	fmt.Printf("unknown verb of an Int: %t\n", i)
	fmt.Printf("nil Int: %d\n", (*big.Int)(nil))

	f := new(big.Float).SetFloat64(1.5)
	fmt.Printf("unknown verb of a Float: %t\n", f)

	fmt.Printf("pad an Int: [%20d] [%-20d] [%020d]\n", i, i, i)
	fmt.Printf("pad a Float: [%20.3f] [%-20.3f] [%+.3e]\n", f, f, f)
	fmt.Printf("base of an Int: %b %o %O %x %X %#x\n", i, i, i, i, i, i)

	// A value above 8 words takes the recursive path of nat.itoa, and that
	// path reads the divisor cache that the patch of natconv.go changed.
	big1 := new(big.Int).Lsh(big.NewInt(1), 4096)
	text := big1.String()
	fmt.Println("digits of 2^4096:", len(text))
	fmt.Println("the first 20 digits:", text[:20])
	fmt.Println("the last 20 digits:", text[len(text)-20:])

	back, ok := new(big.Int).SetString(text, 10)
	fmt.Println("the text reads back:", ok, back.Cmp(big1) == 0)

	// The cache grows over 2 conversions of a different size, thus this call
	// reads the table that the call above built.
	big2 := new(big.Int).Lsh(big.NewInt(1), 8192)
	fmt.Println("digits of 2^8192:", len(big2.String()))
	fmt.Println("digits of 2^4096 again:", len(big1.String()))

	fmt.Println("hex of 2^4096:", strings.Count(big1.Text(16), "0"))
}
