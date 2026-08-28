package main

import (
	"fmt"
	"math"
)

func main() {
	fmt.Println(math.Sqrt(16), math.Abs(-3.5), math.Floor(2.7), math.Ceil(2.1))
	fmt.Println(math.Max(1, 2), math.Min(1, 2), math.Trunc(-2.7))
	fmt.Println(math.Pow(2, 10), math.Mod(7, 3))
	fmt.Println(math.MaxInt64, math.MinInt64, math.MaxInt32)
	fmt.Println(math.Float64bits(1.0), math.Float64frombits(4607182418800017408))
	fmt.Println(math.Float32bits(1.0), math.Float32frombits(1065353216))
	fmt.Println(math.IsNaN(math.NaN()), math.IsInf(math.Inf(1), 1))
	fmt.Println(math.Round(2.5), math.Signbit(-0.0))
	fmt.Printf("%.6f %.6f\n", math.Log(math.E), math.Exp(1))
	fmt.Printf("%.6f %.6f\n", math.Sin(0), math.Cos(0))
	fmt.Printf("%.6f\n", math.Hypot(3, 4))
	fmt.Println(math.Pi > 3.14, math.MaxFloat64 > 0)
}
