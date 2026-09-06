// The program proves the special values of a float: NaN, the 2 infinities, a
// negative zero, and a denormal number. It prints each 1 with 4 verbs, and it
// compares each pair.
package main

import (
	"fmt"
	"math"
	"sort"
	"strconv"
)

type item struct {
	name  string
	value float64
}

func values() []item {
	var zero float64
	return []item{
		{"zero", zero},
		{"neg zero", -zero},
		{"one", 1},
		{"neg one", -1},
		{"half", 0.5},
		{"third", 1.0 / 3.0},
		{"inf", math.Inf(1)},
		{"neg inf", math.Inf(-1)},
		{"NaN", math.NaN()},
		{"max", math.MaxFloat64},
		{"smallest normal", 2.2250738585072014e-308},
		{"denormal", math.SmallestNonzeroFloat64},
		{"denormal x 3", math.SmallestNonzeroFloat64 * 3},
		{"1e21", 1e21},
		{"1e20", 1e20},
		{"1e-5", 1e-5},
		{"123456789", 123456789},
		{"pi", math.Pi},
	}
}

func verbs() {
	fmt.Printf("%-16s %-24s %-24s %-24s %s\n", "name", "%v", "%g", "%e", "%f")
	for _, it := range values() {
		fmt.Printf("%-16s %-24v %-24g %-24e %f\n", it.name, it.value, it.value, it.value, it.value)
	}
	fmt.Println("-- with a precision --")
	for _, it := range values() {
		fmt.Printf("%-16s %-20.3g %-20.3e %-20.3f %.0f\n",
			it.name, it.value, it.value, it.value, it.value)
	}
	fmt.Println("-- other verbs --")
	for _, it := range values() {
		fmt.Printf("%-16s %-24E %-24G %-14x %s\n",
			it.name, it.value, it.value, it.value, strconv.FormatFloat(it.value, 'b', -1, 64))
	}
}

// halfWay proves the rule of a tie: the printer rounds to the even digit.
func halfWay() {
	cases := []float64{0.5, 1.5, 2.5, 3.5, -0.5, -1.5, -2.5, 0.05, 0.15, 0.25,
		0.35, 2.675, 1.005, 8.5, 9.5}
	for _, v := range cases {
		fmt.Printf("%-8g %%.0f=%-6.0f %%.1f=%-8.1f %%.2f=%-9.2f Round=%-6g Trunc=%g\n",
			v, v, v, v, math.Round(v), math.Trunc(v))
	}
}

func compare() {
	nan := math.NaN()
	inf := math.Inf(1)
	ninf := math.Inf(-1)
	var zero float64
	nzero := -zero
	fmt.Println("-- NaN --")
	fmt.Println("nan == nan", nan == nan)
	fmt.Println("nan != nan", nan != nan)
	fmt.Println("nan < 1   ", nan < 1)
	fmt.Println("nan > 1   ", nan > 1)
	fmt.Println("nan <= nan", nan <= nan)
	fmt.Println("IsNaN     ", math.IsNaN(nan))
	fmt.Println("nan + 1   ", nan+1)
	fmt.Println("nan * 0   ", nan*0)
	fmt.Println("-- zero --")
	fmt.Println("0 == -0   ", zero == nzero)
	fmt.Println("signbit 0 ", math.Signbit(zero))
	fmt.Println("signbit -0", math.Signbit(nzero))
	fmt.Println("1/0       ", 1/zero)
	fmt.Println("1/-0      ", 1/nzero)
	fmt.Println("-- infinity --")
	fmt.Println("inf > max ", inf > math.MaxFloat64)
	fmt.Println("ninf < min", ninf < -math.MaxFloat64)
	fmt.Println("inf == inf", inf == inf)
	fmt.Println("inf + inf ", inf+inf)
	fmt.Println("inf - inf ", inf-inf)
	fmt.Println("inf * 0   ", inf*0)
	fmt.Println("1 / inf   ", 1/inf)
	fmt.Println("-1 / inf  ", -1/inf)
	fmt.Println("IsInf +1  ", math.IsInf(inf, 1))
	fmt.Println("IsInf -1  ", math.IsInf(inf, -1))
	fmt.Println("-- denormal --")
	d := math.SmallestNonzeroFloat64
	fmt.Println("d/2       ", d/2)
	fmt.Println("d*0.5     ", d*0.5)
	fmt.Println("d > 0     ", d > 0)
	fmt.Println("d/2 == 0  ", d/2 == 0)
	mx := math.MaxFloat64
	fmt.Println("max*2     ", mx*2)
	fmt.Println("-max*2    ", -mx*2)
}

// bits proves the exact form of each value.
func bits() {
	for _, it := range values() {
		b := math.Float64bits(it.value)
		fmt.Printf("%-16s %016x back=%v\n", it.name, b, math.Float64frombits(b) == it.value ||
			math.IsNaN(it.value))
	}
}

func float32s() {
	var zero float32
	list := []struct {
		name  string
		value float32
	}{
		{"zero", zero},
		{"neg zero", -zero},
		{"inf", float32(math.Inf(1))},
		{"neg inf", float32(math.Inf(-1))},
		{"NaN", float32(math.NaN())},
		{"max", math.MaxFloat32},
		{"denormal", math.SmallestNonzeroFloat32},
		{"third", float32(1.0 / 3.0)},
		{"pi", float32(math.Pi)},
	}
	for _, it := range list {
		// The verb %x of a float32 stands in conf-blocked/confhexfloat32.
		// Read REPORT-conform.md, item 4.
		fmt.Printf("%-10s %-16v %-16g %-16e %-14G %08x\n",
			it.name, it.value, it.value, it.value, it.value, math.Float32bits(it.value))
	}
	var a float32 = 16777216
	fmt.Println("f32 16777216+1", a+1)
	var b float32 = math.MaxFloat32
	fmt.Println("f32 max*2 var ", b*2)
	var dn float32 = math.SmallestNonzeroFloat32
	fmt.Println("f32 den/2     ", dn/2)
	fmt.Println("f32 third f64 ", float64(float32(1.0/3.0)))
}

// order proves that a sort of a list that holds a NaN leaves the NaN in place
// for the less rule of Go.
func order() {
	list := []float64{3, math.NaN(), 1, math.Inf(-1), 2, math.Inf(1), 0}
	sort.Float64s(list)
	fmt.Println("sorted", list)
	other := []float64{3, 1, 2}
	sort.Float64s(other)
	fmt.Println("sorted plain", other)
}

func parse() {
	inputs := []string{"NaN", "nan", "+Inf", "-inf", "0", "-0", "1e400",
		"1e-400", "4.9406564584124654e-324", "0x1p-1", "1_000.5"}
	for _, in := range inputs {
		v, err := strconv.ParseFloat(in, 64)
		fmt.Printf("%-24q %-24v err=%v\n", in, v, err)
	}
}

func main() {
	fmt.Println("== the verbs ==")
	verbs()
	fmt.Println("== the tie ==")
	halfWay()
	fmt.Println("== the comparison ==")
	compare()
	fmt.Println("== the bits ==")
	bits()
	fmt.Println("== float32 ==")
	float32s()
	fmt.Println("== the order ==")
	order()
	fmt.Println("== the parse ==")
	parse()
}
