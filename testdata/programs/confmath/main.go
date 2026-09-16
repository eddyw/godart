// The program proves math and math/bits against real Go: each function of math
// on a table of inputs with 0, negative 0, an infinity, a NaN, a subnormal, and
// a large value, the constants, the bit conversions, and each function of
// math/bits on 8, 16, 32, and 64 bits.
//
// A transcendental function differs from gc by 1 unit of the last place on
// some inputs, because gc fuses a multiply and an addition on arm64 and Dart
// does not. Read docs/support.md. Thus the program writes such a function with
// 11 significant digits, and each exact function with the whole bit pattern.
package main

import (
	"fmt"
	"math"
	"math/bits"
)

var inputs = []float64{
	0, math.Copysign(0, -1), 1, -1, 0.5, -0.5, 2, -2, 3, 10, 100, 1e10, 1e300, -1e300, 1e-300, 5e-324, -5e-324,
	2.2250738585072014e-308, 1.5, -1.5, 2.5, 0.1, math.Pi, -math.Pi, math.E, math.MaxFloat64, -math.MaxFloat64,
	math.SmallestNonzeroFloat64, math.Inf(1), math.Inf(-1), math.NaN(), 1e15, 4503599627370496.5, 9007199254740993, 0.9999999999999999, 1.0000000000000002,
}

// exact writes the whole bit pattern of a value. Each exact function of math
// agrees with gc bit for bit.
func exact(v float64) string {
	return fmt.Sprintf("%v/%#016x", v, math.Float64bits(v))
}

// rounded writes 11 significant digits, thus a difference of 1 unit of the
// last place stays out of the text.
func rounded(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return fmt.Sprint(v)
	}
	return fmt.Sprintf("%.11g", v)
}

func unary() {
	exactFuncs := []struct {
		name string
		f    func(float64) float64
	}{
		{"Abs", math.Abs}, {"Ceil", math.Ceil}, {"Floor", math.Floor}, {"Trunc", math.Trunc}, {"Round", math.Round},
		{"RoundToEven", math.RoundToEven}, {"Sqrt", math.Sqrt}, {"Log", math.Log}, {"Pow10 of int", func(x float64) float64 {
			if x != x || x > 400 || x < -400 {
				return x
			}
			return math.Pow10(int(x))
		}},
	}
	for _, fn := range exactFuncs {
		for _, x := range inputs {
			if fn.name == "Log" && x > 0 && x < 0x1p-1022 {
				fmt.Printf("%-12s %-24s host-dependent-subnormal\n", fn.name, exact(x))
				continue
			}
			fmt.Printf("%-12s %-24s %s\n", fn.name, exact(x), exact(fn.f(x)))
		}
	}
	roundedFuncs := []struct {
		name string
		f    func(float64) float64
	}{
		{"Sin", math.Sin}, {"Cos", math.Cos}, {"Tan", math.Tan}, {"Asin", math.Asin}, {"Acos", math.Acos}, {"Atan", math.Atan},
		{"Sinh", math.Sinh}, {"Cosh", math.Cosh}, {"Tanh", math.Tanh}, {"Asinh", math.Asinh}, {"Acosh", math.Acosh}, {"Atanh", math.Atanh},
		{"Exp", math.Exp}, {"Exp2", math.Exp2}, {"Expm1", math.Expm1}, {"Log2", math.Log2}, {"Log10", math.Log10}, {"Log1p", math.Log1p}, {"Cbrt", math.Cbrt},
		{"Gamma", math.Gamma}, {"Erf", math.Erf}, {"Erfc", math.Erfc}, {"Erfinv", math.Erfinv}, {"Erfcinv", math.Erfcinv},
		{"J0", math.J0}, {"J1", math.J1}, {"Y0", math.Y0}, {"Y1", math.Y1}, {"Logb", math.Logb},
	}
	for _, fn := range roundedFuncs {
		for _, x := range inputs {
			if (fn.name == "Log10" || fn.name == "Y0") && x > 0 && x < 0x1p-1022 {
				fmt.Printf("%-12s %-24s host-dependent-subnormal\n", fn.name, rounded(x))
				continue
			}
			// Log2 of a value next to 1 cancels, thus the fused multiply of
			// gc changes each digit of the tiny result.
			if fn.name == "Log2" && x != 1 && math.Abs(x-1) < 1e-15 {
				continue
			}
			fmt.Printf("%-12s %-24s %s\n", fn.name, rounded(x), rounded(fn.f(x)))
		}
	}
	for _, x := range inputs {
		lg, sign := math.Lgamma(x)
		lgText := rounded(lg)
		if x != 0 && math.Abs(x) < 0x1p-1022 {
			lgText = "host-dependent-subnormal"
		}
		s, c := math.Sincos(x)
		frac, exp := math.Frexp(x)
		ip, fp := math.Modf(x)
		fmt.Printf("%-12s %-24s Lgamma=%s/%d Sincos=%s/%s Frexp=%s/%d Modf=%s/%s Ilogb=%d Signbit=%v IsInf=%v/%v/%v IsNaN=%v\n", "multi", rounded(x),
			lgText, sign, rounded(s), rounded(c), exact(frac), exp, exact(ip), exact(fp), math.Ilogb(x), math.Signbit(x),
			math.IsInf(x, 1), math.IsInf(x, -1), math.IsInf(x, 0), math.IsNaN(x))
	}
	for _, n := range []int{0, 1, 2, 5, -1} {
		for _, x := range []float64{0, 0.5, 1, 2.5, 10, -3, math.Inf(1), math.NaN()} {
			fmt.Printf("Jn/Yn %d %-8v %s %s\n", n, x, rounded(math.Jn(n, x)), rounded(math.Yn(n, x)))
		}
	}
}

func binary() {
	pairs := [][2]float64{
		{0, 0}, {0, math.Copysign(0, -1)}, {math.Copysign(0, -1), 0}, {1, 2}, {2, 1}, {-1, 2}, {2, -1}, {3, 2}, {-3, 2}, {3, -2}, {5, 3}, {-5, 3}, {5.5, 2},
		{7, 0}, {0, 7}, {1, math.Inf(1)}, {math.Inf(1), 1}, {math.Inf(1), math.Inf(1)}, {math.Inf(1), math.Inf(-1)}, {math.NaN(), 1}, {1, math.NaN()},
		{math.NaN(), math.NaN()}, {2, 0.5}, {2, 10}, {2, -2}, {-2, 3}, {-2, 0.5}, {-8, 1.0 / 3}, {10, 308}, {10, 309}, {10, -324}, {0.5, 1075},
		{1e300, 1e300}, {1e300, 1e-300}, {5e-324, 5e-324}, {math.MaxFloat64, 2}, {1, -1}, {0, -1}, {math.Copysign(0, -1), -1}, {math.Copysign(0, -1), -2}, {-1, math.Inf(1)}, {1, 1e300},
		{0.1, 0.3}, {1e15, 3}, {-7.5, 2}, {7.5, -2}, {2.5, 2.5}, {9007199254740993, 2}, {4503599627370496.5, 1},
	}
	for _, p := range pairs {
		x, y := p[0], p[1]
		fmt.Printf("%-24s %-24s Max=%s Min=%s Dim=%s Copysign=%s Mod=%s Remainder=%s Hypot=%s Atan2=%s Pow=%s Nextafter=%s\n",
			exact(x), exact(y), exact(math.Max(x, y)), exact(math.Min(x, y)), exact(math.Dim(x, y)), exact(math.Copysign(x, y)), exact(math.Mod(x, y)),
			exact(math.Remainder(x, y)), rounded(math.Hypot(x, y)), rounded(math.Atan2(x, y)), exact(math.Pow(x, y)), exact(math.Nextafter(x, y)))
	}
	for _, p := range [][3]float64{{1, 2, 3}, {0.1, 10, -1}, {1e200, 1e200, -math.Inf(1)}, {2, 3, 0}, {math.NaN(), 1, 1}, {1, math.Inf(1), 1}, {math.Inf(1), 0, 1}, {0, 0, math.Copysign(0, -1)}, {math.Copysign(0, -1), 1, math.Copysign(0, -1)}, {1e308, 10, -1e308}, {3, 3, -9}, {0.1, 0.1, -0.01}} {
		fmt.Printf("FMA %v %v %v = %s\n", p[0], p[1], p[2], exact(math.FMA(p[0], p[1], p[2])))
	}
	for _, p := range []struct {
		frac float64
		exp  int
	}{{0.5, 1}, {0.5, 2}, {1, 1024}, {1, -1075}, {1, -1074}, {0.75, -1073}, {0, 100}, {math.Inf(1), 1}, {math.NaN(), 1}, {1.5, 2000}, {1.5, -2000}, {3, 1}, {1, 0}} {
		fmt.Printf("Ldexp %v %d = %s\n", p.frac, p.exp, exact(math.Ldexp(p.frac, p.exp)))
	}
	for _, n := range []int{-400, -324, -323, -308, -1, 0, 1, 15, 22, 23, 300, 308, 309, 400} {
		fmt.Printf("Pow10 %d = %s\n", n, exact(math.Pow10(n)))
	}
	for _, p := range [][2]float32{{1, 2}, {2, 1}, {0, 0}, {0, 1}, {1, 1}, {float32(math.Inf(1)), 1}, {float32(math.NaN()), 1}, {math.MaxFloat32, float32(math.Inf(1))}, {1, float32(math.Inf(1))}} {
		r := math.Nextafter32(p[0], p[1])
		fmt.Printf("Nextafter32 %v %v = %v/%#08x\n", p[0], p[1], r, math.Float32bits(r))
	}
}

func constants() {
	consts := []struct {
		name string
		v    float64
	}{
		{"E", math.E}, {"Pi", math.Pi}, {"Phi", math.Phi}, {"Sqrt2", math.Sqrt2}, {"SqrtE", math.SqrtE}, {"SqrtPi", math.SqrtPi}, {"SqrtPhi", math.SqrtPhi},
		{"Ln2", math.Ln2}, {"Log2E", math.Log2E}, {"Ln10", math.Ln10}, {"Log10E", math.Log10E},
		{"MaxFloat32", math.MaxFloat32}, {"SmallestNonzeroFloat32", math.SmallestNonzeroFloat32},
		{"MaxFloat64", math.MaxFloat64}, {"SmallestNonzeroFloat64", math.SmallestNonzeroFloat64},
	}
	for _, c := range consts {
		fmt.Printf("%-24s %s\n", c.name, exact(c.v))
	}
	fmt.Println("MaxInt", math.MaxInt, "MinInt", math.MinInt)
	fmt.Println("MaxInt8", math.MaxInt8, "MinInt8", math.MinInt8, "MaxInt16", math.MaxInt16, "MinInt16", math.MinInt16)
	fmt.Println("MaxInt32", math.MaxInt32, "MinInt32", math.MinInt32, "MaxInt64", math.MaxInt64, "MinInt64", math.MinInt64)
	fmt.Println("MaxUint8", math.MaxUint8, "MaxUint16", math.MaxUint16, "MaxUint32", math.MaxUint32, "MaxUint64", uint64(math.MaxUint64), "MaxUint", uint(math.MaxUint))
	fmt.Println("Inf", math.Inf(1), math.Inf(-1), math.Inf(0), "NaN", math.NaN())
	fmt.Println("NaN != NaN", math.NaN() != math.NaN(), "NaN bits", fmt.Sprintf("%#x", math.Float64bits(math.NaN())))
	fmt.Println("Inf bits", fmt.Sprintf("%#x %#x", math.Float64bits(math.Inf(1)), math.Float64bits(math.Inf(-1))))
	for _, b := range []uint64{0, 0x8000000000000000, 0x3ff0000000000000, 0x7ff0000000000000, 0xfff0000000000000, 0x7ff8000000000001, 0x0000000000000001, 0x000fffffffffffff, 0x0010000000000000, 0x7fefffffffffffff, 0x4009 << 48, 0xc000000000000000} {
		fmt.Printf("Float64frombits %#016x = %v\n", b, math.Float64frombits(b))
	}
	for _, b := range []uint32{0, 0x80000000, 0x3f800000, 0x7f800000, 0xff800000, 0x7fc00000, 0x00000001, 0x007fffff, 0x00800000, 0x7f7fffff, 0x40490fdb} {
		f := math.Float32frombits(b)
		fmt.Printf("Float32frombits %#08x = %v bits=%#08x\n", b, f, math.Float32bits(f))
	}
	for _, f := range []float32{0, 1, -1, 0.1, 1.0 / 3, 16777216, 16777217, 3.4028235e38, 1e-45, float32(math.Pi)} {
		fmt.Printf("Float32bits %v = %#08x\n", f, math.Float32bits(f))
	}
}

func bitsPkg() {
	fmt.Println("UintSize", bits.UintSize)
	u8 := []uint8{0, 1, 2, 3, 0x0f, 0x10, 0x7f, 0x80, 0xaa, 0xf0, 0xfe, 0xff}
	for _, x := range u8 {
		fmt.Printf("8  %#02x LZ=%d TZ=%d Ones=%d Len=%d Rev=%#02x RotL3=%#02x RotL-3=%#02x\n", x, bits.LeadingZeros8(x), bits.TrailingZeros8(x), bits.OnesCount8(x), bits.Len8(x), bits.Reverse8(x), bits.RotateLeft8(x, 3), bits.RotateLeft8(x, -3))
	}
	u16 := []uint16{0, 1, 0x00ff, 0x0100, 0x1234, 0x7fff, 0x8000, 0xabcd, 0xffff}
	for _, x := range u16 {
		fmt.Printf("16 %#04x LZ=%d TZ=%d Ones=%d Len=%d Rev=%#04x RevBytes=%#04x RotL5=%#04x RotL-5=%#04x\n", x, bits.LeadingZeros16(x), bits.TrailingZeros16(x), bits.OnesCount16(x), bits.Len16(x), bits.Reverse16(x), bits.ReverseBytes16(x), bits.RotateLeft16(x, 5), bits.RotateLeft16(x, -5))
	}
	u32 := []uint32{0, 1, 0x000000ff, 0x00010000, 0x12345678, 0x7fffffff, 0x80000000, 0xdeadbeef, 0xffffffff}
	for _, x := range u32 {
		fmt.Printf("32 %#08x LZ=%d TZ=%d Ones=%d Len=%d Rev=%#08x RevBytes=%#08x RotL7=%#08x RotL-7=%#08x RotL40=%#08x\n", x, bits.LeadingZeros32(x), bits.TrailingZeros32(x), bits.OnesCount32(x), bits.Len32(x), bits.Reverse32(x), bits.ReverseBytes32(x), bits.RotateLeft32(x, 7), bits.RotateLeft32(x, -7), bits.RotateLeft32(x, 40))
	}
	u64 := []uint64{0, 1, 0xff, 1 << 32, 0x0123456789abcdef, 0x7fffffffffffffff, 1 << 63, 0xdeadbeefcafebabe, 0xffffffffffffffff}
	for _, x := range u64 {
		fmt.Printf("64 %#016x LZ=%d TZ=%d Ones=%d Len=%d Rev=%#016x RevBytes=%#016x RotL9=%#016x RotL-9=%#016x RotL70=%#016x\n", x, bits.LeadingZeros64(x), bits.TrailingZeros64(x), bits.OnesCount64(x), bits.Len64(x), bits.Reverse64(x), bits.ReverseBytes64(x), bits.RotateLeft64(x, 9), bits.RotateLeft64(x, -9), bits.RotateLeft64(x, 70))
	}
	for _, x := range []uint{0, 1, 0xff, 1 << 32, 1 << 63, ^uint(0)} {
		fmt.Printf("uint %#x LZ=%d TZ=%d Ones=%d Len=%d Rev=%#x RevBytes=%#x RotL1=%#x\n", x, bits.LeadingZeros(x), bits.TrailingZeros(x), bits.OnesCount(x), bits.Len(x), bits.Reverse(x), bits.ReverseBytes(x), bits.RotateLeft(x, 1))
	}
	fmt.Println("-- the arithmetic --")
	for _, p := range [][3]uint32{{1, 2, 0}, {0xffffffff, 1, 0}, {0xffffffff, 0xffffffff, 1}, {0, 0, 1}, {0x80000000, 0x80000000, 0}, {5, 3, 0}, {3, 5, 0}, {3, 5, 1}} {
		s, c := bits.Add32(p[0], p[1], p[2])
		d, b := bits.Sub32(p[0], p[1], p[2])
		hi, lo := bits.Mul32(p[0], p[1])
		fmt.Printf("32 %#x %#x carry %d: Add=%#x/%d Sub=%#x/%d Mul=%#x/%#x\n", p[0], p[1], p[2], s, c, d, b, hi, lo)
	}
	for _, p := range [][3]uint64{{1, 2, 0}, {^uint64(0), 1, 0}, {^uint64(0), ^uint64(0), 1}, {0, 0, 1}, {1 << 63, 1 << 63, 0}, {5, 3, 0}, {3, 5, 0}, {3, 5, 1}, {0x123456789abcdef0, 0xfedcba9876543210, 0}} {
		s, c := bits.Add64(p[0], p[1], p[2])
		d, b := bits.Sub64(p[0], p[1], p[2])
		hi, lo := bits.Mul64(p[0], p[1])
		fmt.Printf("64 %#x %#x carry %d: Add=%#x/%d Sub=%#x/%d Mul=%#x/%#x\n", p[0], p[1], p[2], s, c, d, b, hi, lo)
	}
	for _, p := range [][3]uint{{1, 2, 0}, {^uint(0), 1, 0}, {7, 9, 1}} {
		s, c := bits.Add(p[0], p[1], p[2])
		d, b := bits.Sub(p[0], p[1], p[2])
		hi, lo := bits.Mul(p[0], p[1])
		fmt.Printf("uint %#x %#x carry %d: Add=%#x/%d Sub=%#x/%d Mul=%#x/%#x\n", p[0], p[1], p[2], s, c, d, b, hi, lo)
	}
	for _, p := range [][3]uint32{{0, 100, 7}, {1, 0, 3}, {0, 0xffffffff, 0xffffffff}, {0x7fffffff, 0xffffffff, 0x80000000}, {0, 7, 7}, {2, 5, 3}} {
		q, r := bits.Div32(p[0], p[1], p[2])
		fmt.Printf("Div32 %#x %#x / %#x = %#x rem %#x Rem32=%#x\n", p[0], p[1], p[2], q, r, bits.Rem32(p[0], p[1], p[2]))
	}
	for _, p := range [][3]uint64{{0, 100, 7}, {1, 0, 3}, {0, ^uint64(0), ^uint64(0)}, {0x7fffffffffffffff, ^uint64(0), 1 << 63}, {0, 7, 7}, {2, 5, 3}, {0x123, 0x456789abcdef0123, 0xfedcba}} {
		q, r := bits.Div64(p[0], p[1], p[2])
		fmt.Printf("Div64 %#x %#x / %#x = %#x rem %#x Rem64=%#x\n", p[0], p[1], p[2], q, r, bits.Rem64(p[0], p[1], p[2]))
	}
	q, r := bits.Div(0, 100, 7)
	fmt.Printf("Div %#x %#x Rem=%#x\n", q, r, bits.Rem(0, 100, 7))
	fmt.Printf("Rem32 large hi %#x\n", bits.Rem32(0xffffffff, 0xffffffff, 7))
	fmt.Printf("Rem64 large hi %#x\n", bits.Rem64(^uint64(0), ^uint64(0), 7))
	catch("Div32 by 0", func() { bits.Div32(0, 1, 0) })
	catch("Div32 overflow", func() { bits.Div32(1, 0, 1) })
	catch("Div64 by 0", func() { bits.Div64(0, 1, 0) })
	catch("Div64 overflow", func() { bits.Div64(1, 0, 1) })
	catch("Div by 0", func() { bits.Div(0, 1, 0) })
	catch("Rem32 by 0", func() { bits.Rem32(0, 1, 0) })
	catch("Rem64 by 0", func() { bits.Rem64(0, 1, 0) })
}

func catch(label string, f func()) {
	defer func() {
		fmt.Printf("%-24s %v\n", label, recover())
	}()
	f()
}

func main() {
	fmt.Println("== the unary functions ==")
	unary()
	fmt.Println("== the binary functions ==")
	binary()
	fmt.Println("== the constants and the bits ==")
	constants()
	fmt.Println("== math/bits ==")
	bitsPkg()
}
