//go:build dart

// This file patches the file arith_amd64.go of the package math/big.
//
// The real file reads internal/cpu to find the ADX instruction, and the
// surface holds no such package. The patched arith_decl.go takes the pure Go
// path, thus no caller reads the flag.
package big

const hasADX = false
