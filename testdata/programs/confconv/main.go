// The program proves each numeric conversion of Go, the truncation of a float
// toward zero, the wrap of an integer that does not fit the target, and the
// conversion between a string, a byte slice, and a rune slice.
package main

import (
	"fmt"
	"math"
	"strconv"
)

func row(label string, value any) {
	fmt.Printf("%-26s %v\n", label, value)
}

// intPairs runs a value through each signed target and each unsigned target.
// A value that does not fit the target keeps the low bits, which the
// specification defines.
func intPairs() {
	values := []int64{0, 1, -1, 127, 128, 255, 256, 32767, 32768, 65535,
		2147483647, 2147483648, 4294967295, math.MaxInt64, math.MinInt64, -300}
	for _, v := range values {
		fmt.Printf("%-21d i8=%-5d i16=%-7d i32=%-12d u8=%-4d u16=%-6d u32=%-11d u64=%d\n",
			v, int8(v), int16(v), int32(v), uint8(v), uint16(v), uint32(v), uint64(v))
	}
	uvalues := []uint64{0, 1, 255, 256, 65535, 4294967295, math.MaxUint64,
		math.MaxInt64, math.MaxInt64 + 1}
	for _, v := range uvalues {
		fmt.Printf("u%-20d i8=%-5d i16=%-7d i32=%-12d i64=%-21d u8=%-4d u32=%d\n",
			v, int8(v), int16(v), int32(v), int64(v), uint8(v), uint32(v))
	}
}

// floatToInt proves the truncation toward zero of each sign. Each value
// stands inside the range of its target: the Go specification leaves the
// result of a value outside that range to the implementation, and gc itself
// gives 2 answers on 2 machines.
func floatToInt() {
	values := []float64{0, 0.5, -0.5, 1.5, -1.5, 2.5, -2.5, 0.9999, -0.9999,
		126.5, -127.5, 127, -128, 1.0 / 3.0, 99.9999999}
	for _, v := range values {
		fmt.Printf("%-14g int=%-6d i8=%-6d i16=%-8d i32=%-12d i64=%d\n",
			v, int(v), int8(v), int16(v), int32(v), int64(v))
	}
	uvalues := []float64{0, 0.5, 1.5, 254.9, 255, 200.75}
	for _, v := range uvalues {
		fmt.Printf("%-14g u8=%-6d u16=%-8d u32=%-12d uint=%d\n",
			v, uint8(v), uint16(v), uint32(v), uint(v))
	}
	wide := []float64{1e15, -1e15, 2147483647.9, -2147483648.9, 1e18, -1e18,
		4294967295.5}
	for _, v := range wide {
		fmt.Printf("%-14g int=%-21d i64=%d\n", v, int(v), int64(v))
	}
	row("int(1<<62 as f64)", int64(float64(1<<62)))
	f32 := float32(3.75)
	row("int of a f32", int32(f32))
	row("int of -0", int(negZero()))
	row("uint of 2^62", uint64(float64(1<<62)))
	row("uint of 2^62-1", uint64(float64(1<<62)-1))
}

// intToFloat proves the rounding that a conversion to a float performs.
func intToFloat() {
	values := []int64{0, 1, -1, 1 << 24, 1<<24 + 1, 1 << 53, 1<<53 + 1,
		1<<53 + 2, math.MinInt64, 1 << 62, -1 << 62}
	for _, v := range values {
		fmt.Printf("%-21d f32=%-16g f64=%-24g back=%d\n",
			v, float32(v), float64(v), int64(float64(v)))
	}
	// A conversion of a float64 above 2 to the power 63 into a uint64 stands
	// in conf-blocked/conffloattouint. Read REPORT-conform.md, item 2.
	uvalues := []uint64{0, 1, 1 << 53, 1<<53 + 1, math.MaxUint64, 1 << 63,
		1<<62 + 1}
	for _, v := range uvalues {
		fmt.Printf("u%-20d f32=%-16g f64=%-24g\n", v, float32(v), float64(v))
	}
}

// floatWidth proves the rounding of a float64 into a float32.
func floatWidth() {
	values := []float64{0.1, 1.0 / 3.0, 1e40, -1e40, 1e-45, 1e-46, 3.4028235e38,
		3.4028236e38, math.Pi, 16777217, 16777216}
	for _, v := range values {
		f := float32(v)
		fmt.Printf("%-24g f32=%-16g back=%-24g equal=%v\n", v, f, float64(f), float64(f) == v)
	}
}

// text proves each conversion between a string, a byte slice, and a rune
// slice, and the conversion of an integer to a string.
func text() {
	inputs := []string{"", "a", "hello", "héllo", "日本語", "a\x80b", "\xff\xfe"}
	for _, s := range inputs {
		b := []byte(s)
		r := []rune(s)
		// A width over a slice stands in conf-blocked/conffmtwidth. Read
		// REPORT-conform.md, item 3.
		fmt.Printf("%-10q len=%-2d bytes=%v runes=%v back1=%-10q back2=%-10q\n",
			s, len(s), b, r, string(b), string(r))
	}
	row("rune 65", string(rune(65)))
	row("rune 0x4E2D", string(rune(0x4E2D)))
	row("rune -1", string(rune(-1)))
	row("rune 0x110000", string(rune(0x110000)))
	row("rune 0xD800", string(rune(0xD800)))
	row("rune 0", strconv.Quote(string(rune(0))))
	row("Itoa", strconv.Itoa(-42))
	row("FormatInt base 16", strconv.FormatInt(255, 16))
	row("FormatUint base 2", strconv.FormatUint(5, 2))
	// A byte slice and a string share no storage in Go, thus a write of the
	// slice does not reach the string.
	s := "abc"
	b := []byte(s)
	b[0] = 'z'
	row("string after write", s)
	row("slice after write", string(b))
	// A rune slice of a text with a byte that carries no rune holds the
	// replacement rune, and the round trip does not give the input again.
	bad := "a\xffb"
	back := string([]rune(bad))
	row("bad round trip", strconv.Quote(back))
	row("bad equal", back == bad)
}

// named proves a conversion between a named type and its underlying type.
type myInt int32

type myByte byte

type myText string

type myBytes []byte

type myRunes []rune

func named() {
	var a myInt = 300
	row("myInt", a)
	row("myInt to i8", int8(a))
	row("myInt to myByte", myByte(a))
	row("myInt to f64", float64(a))
	var t myText = "héllo"
	row("myText to bytes", []byte(t))
	row("myText to myBytes", myBytes(t))
	row("myBytes to myText", myText(myBytes("xy")))
	row("myRunes of myText", myRunes(t))
	row("myText of myRunes", myText(myRunes{72, 105}))
	row("len of myText", len(t))
}

func negZero() float64 {
	var z float64
	return -z
}

func main() {
	fmt.Println("== integer to integer ==")
	intPairs()
	fmt.Println("== float to integer ==")
	floatToInt()
	fmt.Println("== integer to float ==")
	intToFloat()
	fmt.Println("== float width ==")
	floatWidth()
	fmt.Println("== text ==")
	text()
	fmt.Println("== named ==")
	named()
}
