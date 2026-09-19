//go:build dart

// Dart output has no native Go module image for an integrity check.
// The FIPS module rejects non-default modes before this package can run.
package check

// Verified reports whether the check of the integrity passed. The FIPS 140-3
// mode never turns on under gart, thus it is always false.
var Verified bool
