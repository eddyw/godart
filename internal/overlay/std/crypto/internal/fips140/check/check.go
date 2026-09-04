//go:build dart

// This file patches the file check.go of the package
// crypto/internal/fips140/check.
//
// The real file reads the symbol go:fipsinfo of the linker through the package
// unsafe, and it hashes the code and the data of the module. gart writes no Go
// binary, and the GODEBUG setting fips140 always holds its default, thus the
// FIPS 140-3 mode never turns on. The init function of the real file returns at
// once in that state, and Verified stays false.
package check

// Verified reports whether the check of the integrity passed. The FIPS 140-3
// mode never turns on under gart, thus it is always false.
var Verified bool
