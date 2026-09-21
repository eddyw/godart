//go:build dart

// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package testenv adapts Go test facilities to the Dart runtime and an optional Go SDK.
package testenv

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
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

// MustHaveSource requires the source tree of the supplied Go SDK.
func MustHaveSource(t testing.TB) {
	t.Helper()
	root := GOROOT(t)
	info, err := os.Stat(filepath.Join(root, "src"))
	if err != nil || !info.IsDir() {
		t.Skipf("testenv: Go SDK source unavailable: %v", err)
	}
}

func MustHaveGoBuild(t testing.TB) {
	t.Helper()
	if os.Getenv("GO_GCFLAGS") != "" {
		t.Skip("testenv: Go build does not support GO_GCFLAGS in this adapter")
	}
	tool := GoToolPath(t)
	if out, err := exec.Command(tool, "tool", "-n", "compile").CombinedOutput(); err != nil {
		t.Skipf("testenv: Go compiler unavailable: %v\n%s", err, out)
	}
}

func MustHaveGoRun(t testing.TB) { MustHaveGoBuild(t) }

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

// GoToolPath selects the executable from the supplied SDK.
func GoToolPath(t testing.TB) string {
	t.Helper()
	name := "go"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	tool, err := exec.LookPath(filepath.Join(GOROOT(t), "bin", name))
	if err != nil {
		t.Skipf("testenv: Go command unavailable: %v", err)
	}
	return tool
}

// MustHaveExec marks a test that starts a process. gart starts a process, thus
// the call skips no test.
func MustHaveExec(t testing.TB) {}

// Executable requires a launcher that the test harness can start again.
func Executable(t testing.TB) string {
	t.Helper()
	path, err := os.Executable()
	if err != nil || path != os.Args[0] {
		t.Skip("testenv: no self-executable test launcher")
	}
	return path
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

func HasExternalNetwork() bool {
	return !testing.Short() && runtime.GOOS != "js" && runtime.GOOS != "wasip1"
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

var execPaths sync.Map // path -> error

// MustHaveExecPath checks that the current system can start the named executable
// using os.StartProcess or (more commonly) exec.Command.
// If not, MustHaveExecPath calls t.Skip with an explanation.
func MustHaveExecPath(t testing.TB, path string) {
	MustHaveExec(t)

	err, found := execPaths.Load(path)
	if !found {
		_, err = exec.LookPath(path)
		err, _ = execPaths.LoadOrStore(path, err)
	}
	if err != nil {
		t.Helper()
		t.Skipf("skipping test: %s: %s", path, err)
	}
}
