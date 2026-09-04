//go:build dart

// This file patches the file alias.go of the package
// crypto/internal/fips140/alias.
//
// The real file compares the address of the first byte and of the last byte of
// each slice, and it reads the package unsafe for that. A slice of gart is a
// view on a Dart list, and the subset names no address, thus no test of the
// overlap stands.
//
// The document of Go says that the destination and the source of a cipher
// operation "must overlap entirely or not at all". A call that breaks that rule
// writes the bytes in order, and it gives no report. Go panics instead. Read
// docs/parity.md.
package alias

// AnyOverlap reports whether x and y share memory. This form always reports
// false.
func AnyOverlap(x, y []byte) bool {
	return false
}

// InexactOverlap reports whether x and y share memory at an index that does not
// match. This form always reports false.
func InexactOverlap(x, y []byte) bool {
	return false
}
