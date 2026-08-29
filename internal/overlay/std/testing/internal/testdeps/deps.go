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
