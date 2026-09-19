// Command mathbits writes the result of each function of math for a wide set
// of inputs. A run under Go and a run under gart must give the same text.
//
// The program holds 2 groups.
//
// The exact group writes the bit pattern of each result. Positive subnormal
// Log and Log10 inputs carry an explicit host-dependent marker.
//
// The near group writes 11 significant digits. Each function of that group
// evaluates a polynomial, and the specification of Go lets an implementation
// fuse a multiply and an addition into 1 operation. gc fuses on arm64 and the
// Dart compiler does not, thus the 2 results differ by 1 unit of the last
// place, and by 2 for Hypot.
package main

import (
	"fmt"
	"math"
)

// values holds each input that a rule of the standard names, and a spread of
// bit patterns that a generator builds.
func values() []float64 {
	out := []float64{
		0,
		math.Copysign(0, -1),
		1, -1, 0.5, -0.5, 2, -2, 3, 4, 16, 100, 1e6,
		0.1, 0.25, 2.5, -2.5, 1.5, -1.5,
		0.49999999999999994,
		4503599627370496, 4503599627370497,
		9007199254740992, 9007199254740993,
		1e-300, 1e300, 1e-16, 1e16,
		math.MaxFloat64,
		math.SmallestNonzeroFloat64,
		math.SmallestNonzeroFloat64 * 3,
		math.Inf(1), math.Inf(-1), math.NaN(),
		math.Pi, -math.Pi, math.E, math.Sqrt2, math.Ln2,
	}
	x := uint64(0x0123456789ABCDEF)
	for i := 0; i < 400; i++ {
		x = x*6364136223846793005 + 1442695040888963407
		out = append(out, math.Float64frombits(x))
	}
	for i := 0; i < 40; i++ {
		out = append(out, float64(i), -float64(i), float64(i)+0.5)
	}
	return out
}

// second holds the right operand of each function of 2 arguments.
var second = []float64{2, -3, 0.5, 0, math.Inf(1), math.NaN()}

func exact1(name string, f func(float64) float64, in []float64) {
	for _, v := range in {
		if (name == "Log" || name == "Log10") && v > 0 && v < 0x1p-1022 {
			fmt.Printf("%s %016x host-dependent-subnormal\n", name, math.Float64bits(v))
			continue
		}
		fmt.Printf("%s %016x %016x\n", name,
			math.Float64bits(v), math.Float64bits(f(v)))
	}
}

func exact2(name string, f func(float64, float64) float64, in []float64) {
	for _, v := range in {
		for _, w := range second {
			fmt.Printf("%s %016x %016x %016x\n", name,
				math.Float64bits(v), math.Float64bits(w),
				math.Float64bits(f(v, w)))
		}
	}
}

func near1(name string, f func(float64) float64, in []float64) {
	for _, v := range in {
		fmt.Printf("%s %016x %.10e\n", name, math.Float64bits(v), f(v))
	}
}

func near2(name string, f func(float64, float64) float64, in []float64) {
	for _, v := range in {
		for _, w := range second {
			fmt.Printf("%s %016x %016x %.10e\n", name,
				math.Float64bits(v), math.Float64bits(w), f(v, w))
		}
	}
}

func main() {
	in := values()

	exact1("Sqrt", math.Sqrt, in)
	exact1("Abs", math.Abs, in)
	exact1("Floor", math.Floor, in)
	exact1("Ceil", math.Ceil, in)
	exact1("Trunc", math.Trunc, in)
	exact1("Round", math.Round, in)
	exact1("RoundToEven", math.RoundToEven, in)
	near1("Exp", math.Exp, in)
	exact1("Exp2", math.Exp2, in)
	exact1("Log", math.Log, in)
	exact1("Log10", math.Log10, in)
	exact1("Log1p", math.Log1p, in)
	exact1("Cbrt", math.Cbrt, in)
	exact1("Acos", math.Acos, in)
	exact1("Atan", math.Atan, in)
	near1("Sinh", math.Sinh, in)
	near1("Cosh", math.Cosh, in)
	near1("Tanh", math.Tanh, in)
	exact2("Pow", math.Pow, in)
	exact2("Mod", math.Mod, in)
	exact2("Max", math.Max, in)
	exact2("Min", math.Min, in)
	exact2("Dim", math.Dim, in)
	exact2("Remainder", math.Remainder, in)
	exact2("Copysign", math.Copysign, in)

	near1("Sin", math.Sin, in)
	near1("Cos", math.Cos, in)
	near1("Tan", math.Tan, in)
	near1("Asin", math.Asin, in)
	near1("Expm1", math.Expm1, in)
	near1("Log2", math.Log2, in)
	near2("Atan2", math.Atan2, in)
	near2("Hypot", math.Hypot, in)
}
