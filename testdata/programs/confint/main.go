// The program proves the arithmetic of each integer type: the wrap of an
// overflow, a shift with a count above the width, a shift of a negative
// value, the sign of a division and of a remainder, the division of the most
// negative value by -1, and the builtins min and max.
package main

import (
	"fmt"
	"math"
)

func row(label string, value any) {
	fmt.Printf("%-22s %v\n", label, value)
}

func rowT(label string, value any) {
	fmt.Printf("%-22s %v %T\n", label, value, value)
}

// The signed section runs the same 8 operations on each signed type.
func signed8() {
	var a, b int8 = 100, 7
	row("i8 a+b", a+b)
	row("i8 a-b", a-b)
	row("i8 a*b", a*b)
	row("i8 a/b", a/b)
	row("i8 a%b", a%b)
	row("i8 overflow", a+a)
	var m int8 = math.MinInt8
	row("i8 min-1", m-1)
	row("i8 -min", -m)
	row("i8 min/-1", m/-1)
	row("i8 min%-1", m%-1)
	row("i8 abs of min", -m)
	rowT("i8 type", a)
}

func signed16() {
	var a, b int16 = 30000, 7
	row("i16 a+b", a+b)
	row("i16 a*b", a*b)
	row("i16 a/b", a/b)
	row("i16 a%b", a%b)
	var m int16 = math.MinInt16
	row("i16 min-1", m-1)
	row("i16 min/-1", m/-1)
	row("i16 min%-1", m%-1)
	rowT("i16 type", a)
}

func signed32() {
	var a, b int32 = 2000000000, 7
	row("i32 a+b", a+b)
	row("i32 a*b", a*b)
	row("i32 a/b", a/b)
	row("i32 a%b", a%b)
	var m int32 = math.MinInt32
	row("i32 min-1", m-1)
	row("i32 min/-1", m/-1)
	row("i32 min%-1", m%-1)
	rowT("i32 type", a)
}

func signed64() {
	var a, b int64 = math.MaxInt64, 7
	row("i64 a+1", a+1)
	row("i64 a*3", a*3)
	row("i64 a/b", a/b)
	row("i64 a%b", a%b)
	var m int64 = math.MinInt64
	row("i64 min-1", m-1)
	row("i64 min/-1", m/-1)
	row("i64 min%-1", m%-1)
	row("i64 min*-1", m*-1)
	rowT("i64 type", a)
}

func unsigned() {
	var a, b uint8 = 200, 7
	var z8 uint8
	row("u8 a+100", a+100)
	row("u8 0-1", z8-1)
	row("u8 a*b", a*b)
	row("u8 a/b", a/b)
	row("u8 a%b", a%b)
	rowT("u8 type", a)

	var c, d uint16 = 60000, 7
	var z16 uint16
	row("u16 c+10000", c+10000)
	row("u16 0-1", z16-1)
	row("u16 c*d", c*d)
	rowT("u16 type", c)

	var e, f uint32 = 4000000000, 7
	var z32 uint32
	row("u32 e+e", e+e)
	row("u32 0-1", z32-1)
	row("u32 e*f", e*f)
	rowT("u32 type", e)

	var g, h uint64 = math.MaxUint64, 7
	var z64 uint64
	row("u64 g+1", g+1)
	row("u64 0-1", z64-1)
	row("u64 g/h", g/h)
	row("u64 g%h", g%h)
	row("u64 g*3", g*3)
	rowT("u64 type", g)

	var p, q uint = math.MaxUint, 3
	row("uint p+1", p+1)
	row("uint p/q", p/q)
	rowT("uint type", p)

	var u uintptr = 1 << 40
	row("uintptr", u)
	rowT("uintptr type", u)
}

// shifts proves the rule of the specification: a count of at least the width
// gives 0 for a left shift and for an unsigned right shift, and it gives 0 or
// -1 for a signed right shift.
func shifts() {
	var x8 uint8 = 0xF0
	for _, n := range []uint{0, 1, 4, 7, 8, 9, 64, 100} {
		fmt.Printf("u8 0xF0<<%-3d %d   >>%-3d %d\n", n, x8<<n, n, x8>>n)
	}
	var s8 int8 = -16
	for _, n := range []uint{0, 1, 4, 7, 8, 9, 64, 100} {
		fmt.Printf("i8 -16<<%-3d %d   >>%-3d %d\n", n, s8<<n, n, s8>>n)
	}
	var s64 int64 = -1
	for _, n := range []uint{0, 1, 62, 63, 64, 65, 200} {
		fmt.Printf("i64 -1<<%-4d %d   >>%-4d %d\n", n, s64<<n, n, s64>>n)
	}
	var u64 uint64 = math.MaxUint64
	for _, n := range []uint{0, 1, 62, 63, 64, 65, 200} {
		fmt.Printf("u64 max<<%-4d %d   >>%-4d %d\n", n, u64<<n, n, u64>>n)
	}
	// A signed count is legal, and a negative count panics.
	var count int = 3
	row("shift by int", int64(1)<<count)
	row("shift neg value", -8>>1)
	row("shift neg value 2", -7>>1)
	row("shift neg value 3", -1>>1)
	var neg = -3
	defer func() {
		row("recover", recover())
	}()
	row("shift by neg", 1<<neg)
}

// divide proves the sign of the quotient and of the remainder. Go truncates
// toward zero, thus the remainder holds the sign of the dividend.
func divide() {
	pairs := [][2]int{{7, 3}, {-7, 3}, {7, -3}, {-7, -3}, {6, 3}, {-6, 3}, {1, 3}, {-1, 3}}
	for _, p := range pairs {
		fmt.Printf("int %3d / %3d = %3d   %% = %3d\n", p[0], p[1], p[0]/p[1], p[0]%p[1])
	}
	upairs := [][2]uint{{7, 3}, {6, 3}, {1, 3}}
	for _, p := range upairs {
		fmt.Printf("uint %3d / %3d = %3d   %% = %3d\n", p[0], p[1], p[0]/p[1], p[0]%p[1])
	}
}

func minmax() {
	row("min int", min(3, 1, 2))
	row("max int", max(3, 1, 2))
	row("min 1 arg", min(5))
	row("max 1 arg", max(5))
	row("min float", min(3.5, 1.5, 2.5))
	row("max float", max(3.5, 1.5, 2.5))
	row("min string", min("pear", "apple", "fig"))
	row("max string", max("pear", "apple", "fig"))
	var a, b int8 = -128, 127
	rowT("min i8", min(a, b))
	rowT("max i8", max(a, b))
	// min and max of a uint64 stand in conf-blocked/confminmaxu64. The Dart
	// output compares the 2 values as signed. Read REPORT-conform.md, item 1.
	var u, v uint32 = 0, math.MaxUint32
	rowT("min u32", min(u, v))
	rowT("max u32", max(u, v))
	// A NaN wins in min and in max.
	nan := math.NaN()
	row("min with NaN", min(1.0, nan))
	row("max with NaN", max(1.0, nan))
	row("min -0 0", math.Signbit(min(negZero(), 0.0)))
	row("max 0 -0", math.Signbit(max(0.0, negZero())))
}

func negZero() float64 {
	var z float64
	return -z
}

// increments proves the ++ and the -- statement and each assign operation.
func increments() {
	var x int8 = 126
	x++
	row("i8 126++", x)
	x++
	row("i8 127++", x)
	x--
	row("i8 -128--", x)
	var y uint8
	y--
	row("u8 0--", y)
	var z int32 = 10
	z += 5
	row("i32 +=", z)
	z -= 20
	row("i32 -=", z)
	z *= 3
	row("i32 *=", z)
	z /= 4
	row("i32 /=", z)
	z %= 4
	row("i32 %%=", z)
	var w uint16 = 0xFF00
	w <<= 4
	row("u16 <<=", w)
	w >>= 8
	row("u16 >>=", w)
	w |= 0x000F
	row("u16 |=", w)
	w &= 0x00FF
	row("u16 &=", w)
	w ^= 0xFFFF
	row("u16 ^=", w)
	w &^= 0xFF00
	row("u16 &^=", w)
}

func main() {
	fmt.Println("== signed ==")
	signed8()
	signed16()
	signed32()
	signed64()
	fmt.Println("== unsigned ==")
	unsigned()
	fmt.Println("== divide ==")
	divide()
	fmt.Println("== min and max ==")
	minmax()
	fmt.Println("== increment ==")
	increments()
	fmt.Println("== shift ==")
	shifts()
	fmt.Println("done")
}
