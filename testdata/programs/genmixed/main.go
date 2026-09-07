// The program proves an arithmetic operator on a type parameter whose set
// mixes a narrow integer type with a float type. The Dart bound of such a set
// is num, and the dictionary of the type argument holds the mask of its width:
// an integer term wraps, float32 rounds to 32 bits, and float64 keeps the
// value. Each value below wraps or rounds, thus a missing mask changes a line.
package main

import (
	"fmt"
	"math"
)

// Number holds the set of golang.org/x/exp/constraints.Integer | Float, less
// the widths that the 6 below already prove.
type Number interface {
	~int8 | ~uint8 | ~int32 | ~uint64 | ~float32 | ~float64
}

type Level int8

type Celsius float32

func row(label string, value any) {
	fmt.Printf("%-24s %v\n", label, value)
}

func Sum[T Number](xs []T) T {
	var total T
	for _, x := range xs {
		total += x
	}
	return total
}

func Max[T Number](a, b T) T {
	if a > b {
		return a
	}
	return b
}

func Min[T Number](a, b T) T {
	if a < b {
		return a
	}
	return b
}

func Abs[T Number](x T) T {
	if x < 0 {
		return -x
	}
	return x
}

func Clamp[T Number](x, lo, hi T) T {
	return Min(Max(x, lo), hi)
}

func Scale[T Number](x, k T) T {
	return x * k
}

// Next adds 1 with the increment statement, thus the constant takes the type
// of the parameter.
func Next[T Number](x T) T {
	x++
	return x
}

func main() {
	row("int8 sum wraps", Sum([]int8{100, 100}))
	row("int8 max", Max[int8](-128, 127))
	row("int8 min", Min[int8](-128, 127))
	row("int8 abs of the minimum", Abs[int8](-128))
	row("int8 abs", Abs[int8](-5))
	row("int8 clamp", Clamp[int8](100, -10, 10))
	row("int8 scale wraps", Scale[int8](100, 3))
	row("int8 next wraps", Next[int8](127))
	row("Level sum wraps", Sum([]Level{100, 100}))

	row("uint8 sum wraps", Sum([]uint8{200, 100}))
	row("uint8 max", Max[uint8](0, 255))
	row("uint8 min", Min[uint8](1, 255))
	row("uint8 abs", Abs[uint8](200))
	row("uint8 clamp", Clamp[uint8](200, 10, 100))
	row("uint8 scale wraps", Scale[uint8](200, 2))
	row("uint8 next wraps", Next[uint8](255))

	row("int32 sum wraps", Sum([]int32{math.MaxInt32, 1}))
	row("int32 max", Max[int32](math.MinInt32, 0))
	row("int32 min", Min[int32](math.MinInt32, 0))
	row("int32 abs of the minimum", Abs[int32](math.MinInt32))
	row("int32 clamp", Clamp[int32](5, -3, 3))
	row("int32 scale wraps", Scale[int32](65536, 65536))
	row("int32 next wraps", Next[int32](math.MaxInt32))

	row("uint64 sum wraps", Sum([]uint64{1 << 63, 1 << 63}))
	row("uint64 sum wraps to 1", Sum([]uint64{math.MaxUint64, 2}))
	row("uint64 max is unsigned", Max[uint64](1<<63, 5))
	row("uint64 min is unsigned", Min[uint64](1<<63, 5))
	row("uint64 abs", Abs[uint64](1<<63))
	row("uint64 clamp is unsigned", Clamp[uint64](1<<63, 0, 1<<62))
	row("uint64 scale wraps", Scale[uint64](1<<32, 1<<32))
	row("uint64 next wraps", Next[uint64](math.MaxUint64))

	row("float32 sum rounds", Sum([]float32{16777216, 1, 1}))
	row("float32 sum of tenths", Sum([]float32{0.1, 0.2}))
	row("float32 max", Max[float32](1.5, 2.5))
	row("float32 min", Min[float32](-1.5, 2.5))
	row("float32 abs", Abs[float32](-2.5))
	row("float32 clamp", Clamp[float32](0.1, 0.25, 0.75))
	row("float32 scale rounds", Scale[float32](0.1, 3))
	row("float32 next rounds", Next[float32](16777216))
	row("Celsius scale rounds", Scale(Celsius(0.1), 3))

	row("float64 sum", Sum([]float64{0.1, 0.2}))
	row("float64 max", Max(1.5, 2.5))
	row("float64 min", Min(-1.5, 2.5))
	row("float64 abs", Abs(-0.5))
	row("float64 clamp", Clamp(0.1, 0.25, 0.75))
	row("float64 scale", Scale(0.1, 3))
	row("float64 next", Next(0.5))
}
