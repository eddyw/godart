package main

import (
	"fmt"
	"math/big"
)

func main() {
	a, _ := new(big.Int).SetString("123456789012345678901234567890", 10)
	b := big.NewInt(987654321)
	fmt.Println("sum:", new(big.Int).Add(a, b))
	fmt.Println("product:", new(big.Int).Mul(a, b))
	q, r := new(big.Int).QuoRem(a, b, new(big.Int))
	fmt.Println("quotient:", q, "remainder:", r)
	fmt.Println("power:", new(big.Int).Exp(big.NewInt(2), big.NewInt(128), nil))
	fmt.Println("hex:", a.Text(16))
	fmt.Println("probably prime:", big.NewInt(104729).ProbablyPrime(20))

	f := new(big.Float).SetPrec(200)
	f.SetString("3.14159265358979323846264338327950288419716939937510")
	fmt.Println("float:", f.Text('f', 40))

	rat := big.NewRat(22, 7)
	fmt.Println("rat:", rat, "float:", rat.FloatString(20))
}
