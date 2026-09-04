//go:build dart

// Package linkhook carries a name of 1 standard library package to another one
// that the import rule of Go keeps apart.
//
// Go reaches such a name with a go:linkname directive, and the subset holds no
// such directive. The package that owns the name writes the field at the start
// of the program, and it holds no import of the reader in the other direction.
// A field that stays nil names a package that the program does not hold.
package linkhook

import "io"

// CheckParallel panics when a test or an ancestor of it called Parallel. The
// argument is a *testing.T, and the package testing writes the field.
var CheckParallel func(t any)

// SetCryptoReader replaces the source of each implicit read of a random value
// of the crypto packages. The package crypto/internal/rand writes the field.
var SetCryptoReader func(r io.Reader)

// SynctestTest runs the body of a synctest bubble under a test of its own, and
// it reports whether that test passed. The package testing writes the field.
//
// The first argument is the *testing.T of the test that holds the bubble, and
// the argument of f is the *testing.T of the bubble.
var SynctestTest func(t any, f func(any)) bool
