//go:build dart

package main

// Each declaration below names 1 Dart symbol. The go command rejects a
// function with no body, thus decl_go.go holds the Go form behind the opposite
// build tag.

//gart:dart dart:math max
func dartMax(a, b float64) float64

//gart:dart dart:math sqrt
func dartSqrt(x float64) float64

//gart:dart dart:core int.parse
func dartParseInt(s string) int

//gart:dart dart:core bool.parse
func dartParseBool(s string) bool

//gart:dart dart:core String.fromCharCode
func dartFromCharCode(c int) string

//gart:dart dart:core print
func dartPrint(s string)
