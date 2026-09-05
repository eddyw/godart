// Command intfloat writes the bit pattern of each conversion between an
// integer type and a float type. A run under Go and a run under gart must give
// the same text.
//
// A value above 2^53 loses bits in a float64, and the rounding must match. The
// unsigned form of 64 bits also reads as a negative Dart int, thus it takes a
// path of its own.
package main

import (
	"fmt"
	"math"
)

func ints() []int64 {
	out := []int64{
		0, 1, -1, 2, -2, 255, 256, -255,
		1 << 20, 1 << 52, 1<<53 - 1, 1 << 53, 1<<53 + 1, 1<<53 + 2,
		1 << 54, 1<<54 + 1, 1<<54 + 3,
		math.MaxInt64, math.MinInt64, math.MaxInt64 - 1, math.MinInt64 + 1,
		math.MaxInt32, math.MinInt32, math.MaxInt32 + 1,
		-(1 << 53), -(1<<53 + 1), -(1 << 62), 1 << 62,
	}
	x := uint64(0x0123456789ABCDEF)
	for i := 0; i < 300; i++ {
		x = x*6364136223846793005 + 1442695040888963407
		out = append(out, int64(x))
	}
	return out
}

func floats() []float64 {
	return []float64{
		0, math.Copysign(0, -1), 1, -1, 0.5, -0.5, 2.5, -2.5,
		1e15, -1e15, 1<<53 - 1, 1 << 53, 1 << 62,
		9.2233720368547758e18, -9.2233720368547758e18,
		1e300, -1e300, 1e-300,
		math.MaxFloat64, math.SmallestNonzeroFloat64,
	}
}

// small holds each value that an int32 can carry. A value outside that range
// gives a result that the specification of Go leaves to the implementation,
// and gc answers 1 way on arm64 and another way on amd64.
func small() []float64 {
	return []float64{
		0, math.Copysign(0, -1), 1, -1, 0.5, -0.5, 2.5, -2.5,
		2147483647, -2147483648, 2147483646.9, -2147483647.9,
		1e9, -1e9, 65536.5, -65536.5,
	}
}

func main() {
	for _, v := range ints() {
		u := uint64(v)
		fmt.Printf("i64 %016x %016x\n", u, math.Float64bits(float64(v)))
		fmt.Printf("u64 %016x %016x\n", u, math.Float64bits(float64(u)))
		fmt.Printf("i32 %016x %016x\n", u, math.Float64bits(float64(int32(v))))
		fmt.Printf("u32 %016x %016x\n", u, math.Float64bits(float64(uint32(u))))
		fmt.Printf("i16 %016x %016x\n", u, math.Float64bits(float64(int16(v))))
		fmt.Printf("u8 %016x %016x\n", u, math.Float64bits(float64(uint8(u))))
		fmt.Printf("f32i64 %016x %08x\n", u, math.Float32bits(float32(v)))
		fmt.Printf("f32u64 %016x %08x\n", u, math.Float32bits(float32(u)))
	}
	for _, f := range floats() {
		b := math.Float64bits(f)
		fmt.Printf("f2i64 %016x %016x\n", b, uint64(int64(f)))
		fmt.Printf("f2f32 %016x %08x\n", b, math.Float32bits(float32(f)))
	}
	for _, f := range small() {
		b := math.Float64bits(f)
		fmt.Printf("f2i32 %016x %08x\n", b, uint32(int32(f)))
		fmt.Printf("f2u32 %016x %08x\n", b, uint32(uint32(int64(f))))
	}
}
