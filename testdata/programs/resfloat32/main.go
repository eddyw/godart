// The program proves the single rounding of a 32-bit float.
//
// Go reads a decimal into 32 bits with 1 rounding, and it writes the shortest
// decimal that reads back as the same 32-bit value. A read that gives a
// 64-bit value first and narrows it rounds 2 times, thus a decimal that
// stands exactly between 2 neighbour values of float32 takes the wrong
// neighbour and the text then holds 1 digit too many.
//
// TestTokenAccessors of encoding/json/jsontext names the 2 texts below.
package main

import (
	"fmt"
	"math"
	"strconv"
)

func row(label string, value any) {
	fmt.Printf("%-26s %v\n", label, value)
}

// texts holds each shape of the grammar, and the 2 witnesses of the double
// rounding at the end.
var texts = []string{
	"0", "-0", "1", "-1", "0.5", "1e10", "1e-10",
	"3.4028235e38", "3.4028236e38", "3.4028234663852886e38",
	"1.4012984643248171e-45", "7.0064923216240854e-46",
	"1e39", "-1e39", "1e-46", "-1e-46",
	"16777215", "16777216", "16777217", "16777218",
	"0.1", "0.2", "0.3", "1.0000001", "1.00000001",
	"9000000000.0000001", "7.038531e-26", "-7.038531e-26",
	"2.1019476964872256063855943749348741969204e-45",
	"2.1019476964872256063e-45",
	"7.0064923216240853546186479164495806564013e-46",
	"0x1p-149", "0x1.fffffep127", "0x1p128",
	"1_000.5", "+2.5", "  3.5",
}

// bits gives the pattern of a 32-bit value, and it names an error.
func bits(text string) string {
	v, err := strconv.ParseFloat(text, 32)
	if err != nil {
		return fmt.Sprintf("err %v", err)
	}
	return fmt.Sprintf("%d", math.Float32bits(float32(v)))
}

// step walks the 32-bit values, and it proves 2 rules of each one: the
// shortest text reads back as the same value, and no text of fewer digits
// reads back. The exponent form gives the count of the digits with no zero of
// the fixed form in the way.
func step() (int, int, string) {
	seed := uint32(12345)
	trips := 0
	shortest := 0
	first := ""
	for i := 0; i < 20000; i++ {
		seed = seed*1103515245 + 12345
		v := math.Float32frombits(seed)
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			continue
		}
		text := strconv.FormatFloat(float64(v), 'e', -1, 32)
		back, err := strconv.ParseFloat(text, 32)
		if err == nil && float32(back) == v {
			trips++
		} else if first == "" {
			first = fmt.Sprintf("round trip %d %s", seed, text)
		}
		least := 9
		for p := 0; p < 9; p++ {
			short := strconv.FormatFloat(float64(v), 'e', p, 32)
			if w, err := strconv.ParseFloat(short, 32); err == nil && float32(w) == v {
				least = p + 1
				break
			}
		}
		// The mantissa of the exponent form holds 1 digit, the point, and the
		// rest. A single digit holds no point.
		digits := 1
		if index := indexByte(text, '.'); index >= 0 {
			digits = indexByte(text, 'e') - index
		}
		if digits == least {
			shortest++
		} else if first == "" {
			first = fmt.Sprintf("shortest %d %s %d %d", seed, text, digits, least)
		}
	}
	return trips, shortest, first
}

// indexByte gives the first place of a byte, or -1.
func indexByte(text string, b byte) int {
	for i := 0; i < len(text); i++ {
		if text[i] == b {
			return i
		}
	}
	return -1
}

func main() {
	for _, text := range texts {
		row(text, bits(text))
	}
	for _, text := range texts {
		v, err := strconv.ParseFloat(text, 32)
		if err != nil {
			continue
		}
		f := float32(v)
		row("text of "+text, fmt.Sprintf("%v %s %s",
			f,
			strconv.FormatFloat(float64(f), 'g', -1, 32),
			strconv.FormatFloat(float64(f), 'e', -1, 32)))
	}
	trips, shortest, first := step()
	row("round trips", trips)
	row("shortest texts", shortest)
	row("first difference", first)
}
