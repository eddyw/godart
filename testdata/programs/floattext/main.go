// Command floattext writes the text of many float values, in each verb and
// each precision that strconv.FormatFloat holds. A run under Go and a run
// under gart must give the same text.
package main

import (
	"fmt"
	"math"
	"strconv"
)

func values() []float64 {
	out := []float64{
		0, math.Copysign(0, -1), 1, -1, 0.5, -0.5, 2.5, -2.5,
		0.1, 0.2, 0.3, 1.0 / 3.0, 2.0 / 3.0,
		3.141592653589793, 2.718281828459045,
		1e-5, 1e-4, 1e-3, 1e-1, 1e1, 1e5, 1e20, 1e-20,
		1e21, 1e22, 1e-21, 1e-22,
		123456789, 1234567890123456789,
		0.49999999999999994, 4503599627370497,
		9007199254740992, 9007199254740993,
		math.MaxFloat64, math.SmallestNonzeroFloat64,
		math.MaxFloat32, math.SmallestNonzeroFloat32,
		math.Inf(1), math.Inf(-1), math.NaN(),
		255.5, 0.125, 1024, 1e300, 1e-300,
	}
	x := uint64(0x0123456789ABCDEF)
	for i := 0; i < 120; i++ {
		x = x*6364136223846793005 + 1442695040888963407
		v := math.Float64frombits(x)
		if math.IsNaN(v) {
			continue
		}
		out = append(out, v)
	}
	for i := -30; i <= 30; i++ {
		out = append(out, float64(i)/7.0, float64(i)*1.5)
	}
	return out
}

func main() {
	verbs := []byte{'f', 'F', 'e', 'E', 'g', 'G'}
	precisions := []int{-1, 0, 1, 2, 3, 6, 10, 17}
	for _, v := range values() {
		bits := math.Float64bits(v)
		for _, verb := range verbs {
			for _, p := range precisions {
				fmt.Printf("%016x %c %d 64 %s\n", bits, verb, p,
					strconv.FormatFloat(v, verb, p, 64))
				fmt.Printf("%016x %c %d 32 %s\n", bits, verb, p,
					strconv.FormatFloat(float64(float32(v)), verb, p, 32))
			}
		}
		fmt.Printf("%016x A %s|%s|%s\n", bits,
			strconv.AppendFloat([]byte("a:"), v, 'g', -1, 64),
			strconv.FormatFloat(v, 'b', -1, 64),
			strconv.FormatFloat(v, 'x', -1, 64))
		fmt.Printf("%016x V %v %g %e %f %.3f %.0f %8.2f %-9.3e|\n",
			bits, v, v, v, v, v, v, v, v)
		fmt.Printf("%016x S %s\n", bits, fmt.Sprint(v))
	}
	// A precision above 20 stands outside the reach of toStringAsFixed of
	// Dart, and Go writes each digit of the exact value.
	for _, v := range []float64{0.1, 1.0 / 3.0, 2.5, 1e-8, -0.1, 1e21, 12345.6789} {
		for _, p := range []int{18, 20, 21, 25, 30, 40, 60} {
			fmt.Printf("H %016x %d %s\n", math.Float64bits(v), p,
				strconv.FormatFloat(v, 'f', p, 64))
			fmt.Printf("H %016x %d e %s\n", math.Float64bits(v), p,
				strconv.FormatFloat(v, 'e', p, 64))
		}
	}
}
