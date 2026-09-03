package main

import (
	"fmt"
	"math"
	"strconv"
)

// values holds each shape that the shortest exponent text must hold.
var values = []float64{
	0, 1, -1, 100, 0.1, 1.5, -1.5, 1e21, 1e-7, 1e100, 1e-100,
	12345678901234567890, 0.000001, 123456.789, math.MaxFloat64,
	math.SmallestNonzeroFloat64, math.Inf(1), math.Inf(-1), math.NaN(),
}

func main() {
	for _, v := range values {
		fmt.Println(strconv.FormatFloat(v, 'e', -1, 64),
			strconv.FormatFloat(v, 'E', -1, 64),
			strconv.FormatFloat(v, 'f', -1, 64),
			strconv.FormatFloat(v, 'g', -1, 64))
	}
	for _, v := range []float32{0, 1, 100, 0.1, 1e20, 1e-7} {
		fmt.Println(strconv.FormatFloat(float64(v), 'e', -1, 32))
	}
	fmt.Println(strconv.FormatFloat(1e21, 'e', 3, 64), strconv.FormatFloat(1e21, 'e', 0, 64))
}
