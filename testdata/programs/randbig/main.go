package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// A random value differs on each run, thus this program prints a property of
// each value and never the value itself.
func main() {
	max := new(big.Int).Lsh(big.NewInt(1), 128)
	inRange := true
	for range 50 {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			fmt.Println("Int gave an error:", err)
			return
		}
		if n.Sign() < 0 || n.Cmp(max) >= 0 {
			inRange = false
		}
	}
	fmt.Println("each of 50 values falls in the range:", inRange)

	small, err := rand.Int(rand.Reader, big.NewInt(1))
	fmt.Println("the only value below 1:", small, err)

	for _, bits := range []int{8, 17, 64} {
		p, err := rand.Prime(rand.Reader, bits)
		if err != nil {
			fmt.Println("Prime gave an error:", err)
			return
		}
		fmt.Printf("a prime of %d bits: length %d, prime %t, odd %t\n",
			bits, p.BitLen(), p.ProbablyPrime(20), p.Bit(0) == 1)
	}

	_, err = rand.Prime(rand.Reader, 1)
	fmt.Println("Prime of 1 bit:", err)
}
