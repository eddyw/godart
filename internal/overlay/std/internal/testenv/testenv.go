//go:build dart

// Package testenv gives the part of the internal test helper of Go that the
// tests of the surface read.
//
// The real package starts a process of the go command, and it reads the build
// configuration of the machine. gart holds no such access, thus the overlay
// gives the answer of a local run with each optimization on.
package testenv

import (
	"os"
	"os/exec"
	"runtime"
	"testing"
)

// Builder gives the name of the builder that runs the test. A local run gives
// an empty name, and each caller of the surface reads it for that reason.
func Builder() string { return "" }

// OptimizationOff reports whether the build runs with no optimization. The
// Dart toolchain holds no such mode.
func OptimizationOff() bool { return false }

// SkipIfOptimizationOff skips a test that needs the optimizations of the
// compiler. The Dart toolchain holds no mode with no optimization.
func SkipIfOptimizationOff(t testing.TB) {}

// MustHaveSymlink skips a test when the system cannot make a symbolic link.
// The check makes 1 link in a temporary directory, exactly as Go does on
// Windows.
func MustHaveSymlink(t testing.TB) {
	dir, err := os.MkdirTemp("", "symtest")
	if err != nil {
		t.Skipf("skipping test: cannot make a temporary directory: %v", err)
	}
	defer os.RemoveAll(dir)
	if err := os.Symlink("target", dir+"/link"); err != nil {
		t.Skipf("skipping test: cannot make symlinks: %v", err)
	}
}

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

func HasCGO() bool { return false }

// GOROOT gives the SDK root from the process-start environment.
func GOROOT(t testing.TB) string {
	root := runtime.GOROOT()
	if root == "" && t != nil {
		t.Skip("testenv: no Go SDK root in the process environment")
	}
	return root
}

// ParallelOn64Bit marks a test that runs in parallel on a 64-bit platform. A
// Dart int holds 64 bits, thus the call marks each run.
func ParallelOn64Bit(t *testing.T) { t.Parallel() }

// GoToolPath skips the test. The Dart output holds no go command.
func GoToolPath(t testing.TB) string {
	t.Skip("testenv: the Dart output holds no go command")
	return ""
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

// SkipFlaky skips a test that the Go project marks as flaky. The real form
// runs the test with the flag -flaky, and the test binary of gart holds no
// such flag, thus the call always skips.
func SkipFlaky(t testing.TB, issue int) {
	t.Helper()
	t.Skipf("skipping known flaky test; see golang.org/issue/%d", issue)
}

func CPUIsSlow() bool {
	switch runtime.GOARCH {
	case "arm", "mips", "mipsle", "mips64", "mips64le", "wasm":
		return true
	}
	return false
}

func SkipIfShortAndSlow(t testing.TB) {
	if testing.Short() && CPUIsSlow() {
		t.Helper()
		t.Skipf("skipping test in -short mode on %s", runtime.GOARCH)
	}
}

func MustHaveExternalNetwork(t testing.TB) {
	if runtime.GOOS == "js" || runtime.GOOS == "wasip1" {
		t.Helper()
		t.Skipf("skipping test: no external network on %s", runtime.GOOS)
	}
	if testing.Short() {
		t.Helper()
		t.Skip("skipping test: no external network in -short mode")
	}
}
