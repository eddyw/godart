//go:build dart

// Package testenv gives the part of the internal test helper of Go that the
// tests of the surface read.
//
// The real package starts a process of the go command, and it reads the build
// configuration of the machine. gart holds no such access, thus the overlay
// gives the answer of a local run with each optimization on.
package testenv

import (
	"os/exec"
	"testing"
)

// Builder gives the name of the builder that runs the test. A local run gives
// an empty name, and each caller of the surface reads it for that reason.
func Builder() string { return "" }

// SkipIfOptimizationOff skips a test that needs the optimizations of the
// compiler. The Dart toolchain holds no mode with no optimization.
func SkipIfOptimizationOff(t testing.TB) {}

// MustHaveSource skips the test. The Dart output holds no source tree of Go.
func MustHaveSource(t testing.TB) {
	t.Skip("testenv: the Dart output holds no source tree of Go")
}

// MustHaveGoBuild skips the test. The Dart output holds no go command.
func MustHaveGoBuild(t testing.TB) {
	t.Skip("testenv: the Dart output holds no go command")
}

// MustHaveGoRun skips the test. The Dart output holds no go command.
func MustHaveGoRun(t testing.TB) {
	t.Skip("testenv: the Dart output holds no go command")
}

// MustHaveCGO skips the test. Non-goal N2 of the project holds cgo out.
func MustHaveCGO(t testing.TB) {
	t.Skip("testenv: the subset holds no cgo")
}

// GOROOT gives the root of the Go installation. A test that reads it needs the
// source tree of Go, thus this form skips the test. A caller that gives no
// test reads the empty name.
func GOROOT(t testing.TB) string {
	if t != nil {
		t.Skip("testenv: the Dart output holds no source tree of Go")
	}
	return ""
}

// ParallelOn64Bit marks a test that runs in parallel on a 64-bit platform. A
// Dart int holds 64 bits, thus the call marks each run.
func ParallelOn64Bit(t *testing.T) { t.Parallel() }

// GoToolPath skips the test. The Dart output holds no go command.
func GoToolPath(t testing.TB) string {
	t.Skip("testenv: the Dart output holds no go command")
	return ""
}

// Command skips the test. The Dart output holds no go command, thus a test
// that runs one has no meaning.
func Command(t testing.TB, name string, args ...string) *exec.Cmd {
	t.Skip("testenv: the Dart output runs no command of the go tool chain")
	return nil
}

// MustHaveExec marks a test that starts a process. gart starts a process, thus
// the call skips no test.
func MustHaveExec(t testing.TB) {}

// Executable skips the test. The Dart output is no test binary that starts
// itself again.
func Executable(t testing.TB) string {
	t.Skip("testenv: the Dart output does not start itself again")
	return ""
}

// CleanCmdEnv gives the command back with no change. The real form drops each
// variable of the test environment.
func CleanCmdEnv(cmd *exec.Cmd) *exec.Cmd { return cmd }
