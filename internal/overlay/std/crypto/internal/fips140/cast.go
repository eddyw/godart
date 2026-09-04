//go:build dart

// This file patches the file cast.go of the package crypto/internal/fips140.
//
// The real file reads runtime.fatal through a go:linkname directive. The
// GODEBUG setting fips140 always holds its default under gart, thus Enabled is
// false, CAST and PCT each return at once, and no self test runs. A panic
// stands in place of the call of fatal.
package fips140

import (
	"crypto/internal/fips140deps/godebug"
	"errors"
	"strings"
)

var failfipscast = godebug.Value("#failfipscast")

// CAST runs the named Cryptographic Algorithm Self-Test in the FIPS 140-3
// mode, and it stops the program when the test fails.
func CAST(name string, f func() error) {
	if strings.ContainsAny(name, ",#=:") {
		panic("fips: invalid self-test name: " + name)
	}
	if !Enabled {
		return
	}

	err := f()
	if name == failfipscast {
		err = errors.New("simulated CAST failure")
	}
	if err != nil {
		panic("FIPS 140-3 self-test failed: " + name + ": " + err.Error())
	}
	if debug {
		println("FIPS 140-3 self-test passed:", name)
	}
}

// PCT runs the named Pairwise Consistency Test in the FIPS 140-3 mode, and it
// stops the program when the test fails.
func PCT(name string, f func() error) {
	if strings.ContainsAny(name, ",#=:") {
		panic("fips: invalid self-test name: " + name)
	}
	if !Enabled {
		return
	}

	err := f()
	if name == failfipscast {
		err = errors.New("simulated PCT failure")
	}
	if err != nil {
		panic("FIPS 140-3 self-test failed: " + name + ": " + err.Error())
	}
	if debug {
		println("FIPS 140-3 PCT passed:", name)
	}
}
