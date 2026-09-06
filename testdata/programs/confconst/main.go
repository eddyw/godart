// The program proves the constant rules of Go. A constant expression holds
// arbitrary precision, thus a value that stands above the range of an int64
// gives an exact result. The program also proves iota, a constant shift, a
// typed constant, and a rune constant.
package main

import "fmt"

// The size constants prove iota inside a shift expression.
const (
	kb = 1 << (10 * (iota + 1))
	mb
	gb
	tb
	pb
)

// The flag constants prove iota with a blank name in the middle.
const (
	flagA = 1 << iota
	flagB
	_
	flagD
)

// The pair constants prove 2 names in 1 line of a constant block.
const (
	low, high = iota, iota + 100
	next, over
)

// big stands far above the range of an int64. The division brings the value
// back into the range, thus the program can print it.
const (
	big      = 1 << 200
	huge     = big * big
	backDown = huge / (1 << 390)
	exact    = (1<<62+1)*4/4 - 1<<62
	ratio    = huge / big / (1 << 190)
)

type celsius float64

type weekday int

// The typed constants prove that the type sticks to the name.
const (
	freeze celsius = 0
	boil   celsius = 100
	third  celsius = boil / 3

	maxU8  uint8  = 1<<8 - 1
	minI8  int8   = -1 << 7
	maxI8  int8   = 1<<7 - 1
	maxU16 uint16 = 1<<16 - 1
	maxI32 int32  = 1<<31 - 1
	minI32 int32  = -1 << 31
	maxI64 int64  = 1<<63 - 1
	minI64 int64  = -1 << 63
	maxU64 uint64 = 1<<64 - 1
	maxUin uint   = 1<<64 - 1

	sunday weekday = iota - 12
	monday
	tuesday
)

// The float constants hold more digits than a float64 holds. The conversion
// rounds them 1 time, at the point of the use.
const (
	pi       = 3.14159265358979323846264338327950288419716939937510582097494459
	euler    = 2.71828182845904523536028747135266249775724709369995957496696763
	product  = pi * euler
	oneThird = 1.0 / 3.0
	tiny     = 1.0 / (1 << 60)
	halfWay  = 1<<53 + 1
	intDiv   = 7 / 2
	floatDiv = 7.0 / 2
	mixedDiv = 7 / 2.0
)

// The rune constants prove each escape form.
const (
	nul       = '\x00'
	bell      = '\a'
	tabRune   = '\t'
	quoteRune = '\''
	letterA   = 'A'
	nextChar  = letterA + 1
	euro      = '€'
	maxRune   = '\U0010FFFF'
	replace   = '�'
	octal     = '\101'
)

const (
	greeting = "hello" + ", " + "world"
	raw      = `a	b\n`
	fromRune = string(rune(letterA))
	fromEuro = string(rune(euro))
	textLen  = len(greeting)
)

const (
	yes      = true
	no       = !yes
	compared = pi > euler
	shifted  = maxI64 >> 62
	ushifted = maxU64 >> 62
	andOp    = 0xF0F0 & 0x0FF0
	orOp     = 0xF000 | 0x000F
	xorOp    = 0xFF00 ^ 0x0FF0
	andNot   = 0xFFFF &^ 0x0F0F
	notOp    = ^uint8(0)
	negOp    = -maxI8
)

// The array length reads a constant, thus the constant must be an integer.
var table [kb / 512]int

func row(label string, value any) {
	fmt.Printf("%-14s %v\n", label, value)
}

func main() {
	fmt.Println("== iota ==")
	row("kb", kb)
	row("mb", mb)
	row("gb", gb)
	row("tb", tb)
	row("pb", pb)
	row("flagA", flagA)
	row("flagB", flagB)
	row("flagD", flagD)
	row("low", low)
	row("high", high)
	row("next", next)
	row("over", over)
	row("sunday", int(sunday))
	row("monday", int(monday))
	row("tuesday", int(tuesday))

	fmt.Println("== arbitrary precision ==")
	row("backDown", backDown)
	row("exact", exact)
	row("ratio", ratio)
	row("big/2^190", big/(1<<190))
	row("float(big)", float64(big))
	row("huge float", float64(huge)/float64(big))

	fmt.Println("== typed ==")
	row("freeze", freeze)
	row("boil", boil)
	row("third", third)
	row("third type", fmt.Sprintf("%T", third))
	row("maxU8", maxU8)
	row("minI8", minI8)
	row("maxI8", maxI8)
	row("maxU16", maxU16)
	row("maxI32", maxI32)
	row("minI32", minI32)
	row("maxI64", maxI64)
	row("minI64", minI64)
	row("maxU64", maxU64)
	row("maxUin", maxUin)
	row("u64 type", fmt.Sprintf("%T", maxU64))
	row("i8 type", fmt.Sprintf("%T", minI8))

	fmt.Println("== float ==")
	row("pi", pi)
	row("euler", euler)
	row("product", product)
	row("oneThird", oneThird)
	row("tiny", tiny)
	row("halfWay", halfWay)
	row("halfWay f64", float64(halfWay))
	row("intDiv", intDiv)
	row("floatDiv", floatDiv)
	row("mixedDiv", mixedDiv)
	row("pi float32", float32(pi))
	row("pi type", fmt.Sprintf("%T", pi))
	row("intDiv type", fmt.Sprintf("%T", intDiv))

	fmt.Println("== rune ==")
	row("nul", nul)
	row("bell", bell)
	row("tabRune", tabRune)
	row("quoteRune", quoteRune)
	row("letterA", letterA)
	row("nextChar", nextChar)
	row("euro", euro)
	row("maxRune", maxRune)
	row("replace", replace)
	row("octal", octal)
	row("rune type", fmt.Sprintf("%T", letterA))
	row("as text", string(rune(nextChar)))

	fmt.Println("== string ==")
	row("greeting", greeting)
	row("raw", raw)
	row("raw len", len(raw))
	row("fromRune", fromRune)
	row("fromEuro", fromEuro)
	row("fromEuro len", len(fromEuro))
	row("textLen", textLen)

	fmt.Println("== boolean and bits ==")
	row("yes", yes)
	row("no", no)
	row("compared", compared)
	row("shifted", shifted)
	row("ushifted", ushifted)
	row("andOp", andOp)
	row("orOp", orOp)
	row("xorOp", xorOp)
	row("andNot", andNot)
	row("notOp", notOp)
	row("negOp", negOp)

	fmt.Println("== a constant in a context ==")
	row("table len", len(table))
	var f32 float32 = pi
	var f64 float64 = pi
	row("f32", f32)
	row("f64", f64)
	row("f32 == f64", float64(f32) == f64)
	var small int8 = maxI8
	small++
	row("wrap of maxI8", small)
	row("shift by 0", maxI64>>0)
	row("shift by 63", maxU64>>63)
	row("1<<62 float", float64(1<<62))
	row("complex", complex(3, 4))
	row("complex real", real(3+4i))
	row("complex imag", imag(3+4i))
}
