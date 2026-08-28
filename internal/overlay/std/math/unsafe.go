//go:build dart

// This file patches the file of the same name of the package math.
//
// The real file reinterprets the bytes of a value with the package unsafe. The
// Dart runtime holds 1 buffer with a view of each type, thus the interop
// package gives the same result.
package math

import "syscall/dart"

func Float32bits(f float32) uint32 { return dart.Float32Bits(f) }

func Float32frombits(b uint32) float32 { return dart.Float32FromBits(b) }

func Float64bits(f float64) uint64 { return dart.Float64Bits(f) }

func Float64frombits(b uint64) float64 { return dart.Float64FromBits(b) }
