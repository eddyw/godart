package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

func main() {
	one := big.NewInt(1)
	n, err := rand.Int(rand.Reader, one)
	fmt.Println("the only value below 1:", n, err)

	max, _ := new(big.Int).SetString("1000000000000000000000000000000", 10)
	for i := 0; i < 20; i++ {
		v, err := rand.Int(rand.Reader, max)
		if err != nil {
			fmt.Println("int:", err)
			return
		}
		if v.Sign() < 0 || v.Cmp(max) >= 0 {
			fmt.Println("out of range:", v)
			return
		}
	}
	fmt.Println("20 values each fall in the range")

	func() {
		defer func() { fmt.Println("a max of 0 panics:", recover()) }()
		rand.Int(rand.Reader, big.NewInt(0))
	}()

	for _, bits := range []int{16, 64, 128} {
		p, err := rand.Prime(rand.Reader, bits)
		if err != nil {
			fmt.Println("prime:", err)
			return
		}
		fmt.Printf("a prime of %d bits: len=%d prime=%v odd=%v\n",
			bits, p.BitLen(), p.ProbablyPrime(20), p.Bit(0) == 1)
	}

	if _, err := rand.Prime(rand.Reader, 1); err != nil {
		fmt.Println("1 bit gives:", err)
	}
}
