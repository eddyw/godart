//go:build !dart

package main

import (
	"fmt"
	"math"
	"strconv"
)

func dartMax(a, b float64) float64 { return math.Max(a, b) }

func dartSqrt(x float64) float64 { return math.Sqrt(x) }

func dartParseInt(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func dartParseBool(s string) bool {
	b, _ := strconv.ParseBool(s)
	return b
}

func dartFromCharCode(c int) string { return string(rune(c)) }

func dartPrint(s string) { fmt.Println(s) }
