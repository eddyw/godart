package main

import (
	"fmt"
	"math"
)

type Number interface {
	~int8 | ~int64 | ~uint64 | ~float32 | ~float64
}
type Integer interface{ ~int8 | ~int64 | ~uint64 }

func div[T Number](a, b T) T            { a /= b; return a }
func mod[T Integer](a, b T) T           { a %= b; return a }
func shift[T Integer](a T, n uint) T    { a >>= n; return a }
func convert[T Number, U Number](a T) U { return U(a) }
func compare[T Number](a, b T)          { fmt.Println(a < b, a <= b, a > b, a >= b) }
func intcompare[T Integer](a, b T)      { fmt.Println(a < b, a <= b, a > b, a >= b) }
func printInt[T Integer](a T)           { println(a) }
func main() {
	fmt.Println(div[int8](-128, -1), div[int64](-7, 2), div[uint64](math.MaxUint64, 2))
	fmt.Println(div[float32](7, 2), div[float64](-7, 2))
	fmt.Println(mod[int64](-7, 2), mod[uint64](math.MaxUint64, 3))
	fmt.Println(shift[int64](-8, 2), shift[uint64](1<<63, 63))
	fmt.Println(convert[uint64, float64](math.MaxUint64), convert[float64, int8](130.5))
	fmt.Println(convert[int64, float32](16777217), convert[float64, uint64](math.Exp2(63)))
	compare(math.NaN(), 1.0)
	compare(1.0, math.NaN())
	compare[uint64](1<<63, 1)
	intcompare[uint64](1<<63, 1)
	intcompare[int64](-1, 1)
	printInt[uint64](math.MaxUint64)
}
