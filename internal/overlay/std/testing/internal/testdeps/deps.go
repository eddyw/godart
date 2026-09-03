//go:build dart

// Package testdeps gives the test harness the match function of the package
// regexp.
//
// The package testing imports no regexp, because regexp/syntax imports testing
// in its own test and the 2 imports would give a cycle. The go command puts
// this package in the import list of each test binary, thus the entry point of
// gart reaches it and the cycle stays away.
package testdeps

import (
	"regexp"
	"testing"
)

func init() { testing.MatchString = regexp.MatchString }

// TestDeps holds the calls that a generated main of Go gives the harness. Go
// holds many more calls in this type, and each other one names a profile or a
// fuzz target. Read docs/parity.md.
type TestDeps struct{}

// MatchString matches 1 element of a test name against 1 part of a pattern.
func (TestDeps) MatchString(pat, str string) (bool, error) {
	return regexp.MatchString(pat, str)
}
