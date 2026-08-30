//go:build dart

// Package testenv gives the part of the internal test helper of Go that the
// tests of the surface read.
//
// The real package starts a process of the go command, and it reads the build
// configuration of the machine. gart holds no such access, thus the overlay
// gives the answer of a local run with each optimization on.
package testenv

import "testing"

// Builder gives the name of the builder that runs the test. A local run gives
// an empty name, and each caller of the surface reads it for that reason.
func Builder() string { return "" }

// SkipIfOptimizationOff skips a test that needs the optimizations of the
// compiler. The Dart toolchain holds no mode with no optimization.
func SkipIfOptimizationOff(t testing.TB) {}
