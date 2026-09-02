package main

import "fmt"

func main() {
	fmt.Println("max:", dartMax(2.5, 7.25))
	fmt.Println("sqrt:", dartSqrt(144))
	fmt.Println("parseInt:", dartParseInt("4321"))
	fmt.Println("parseBool:", dartParseBool("true"), dartParseBool("false"))
	fmt.Println("fromCharCode:", dartFromCharCode(65), dartFromCharCode(0x4e2d))
	dartPrint("a line from the Dart library")

	total := 0.0
	for i := 1; i <= 5; i++ {
		total = dartMax(total, dartSqrt(float64(i*i*i)))
	}
	fmt.Printf("total: %.4f\n", total)
}
