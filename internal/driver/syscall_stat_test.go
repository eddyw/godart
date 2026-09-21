//go:build darwin || linux

package driver_test

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestSyscallStatNilOutput(t *testing.T) {
	source, err := filepath.Abs("../../testdata/programs/syscallstat")
	if err != nil {
		t.Fatal(err)
	}
	native := filepath.Join(t.TempDir(), "program")
	run(t, source, "go", "build", "-o", native, ".")
	nativeOutput, err, completed := output(source, nil, runLimit, native, "nil")
	if !completed {
		t.Fatalf("native nil-output reference exceeded %v", runLimit)
	}
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok && runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
			status := exit.Sys().(syscall.WaitStatus)
			if status.Signaled() && status.Signal() == syscall.SIGSEGV {
				t.Skipf("native Linux amd64 Go reference receives SIGSEGV on nil stat output: %s", nativeOutput)
			}
		}
		t.Fatalf("native nil-output reference: %v\n%s", err, nativeOutput)
	}
	want := programResult{Text: cutTrace(string(nativeOutput))}
	out := t.TempDir()
	if err := driver.Build(driver.Options{Dir: source, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Format: false}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	got := programOutcome(t, out, "dart", "bin/main.dart", "nil")
	if got != want {
		t.Errorf("VM nil-output contract differs: got %+v, want %+v", got, want)
	}
	run(t, out, "dart", "compile", "exe", "bin/main.dart", "-o", "app")
	got = programOutcome(t, out, filepath.Join(out, "app"), "nil")
	if got != want {
		t.Errorf("AOT nil-output contract differs: got %+v, want %+v", got, want)
	}
}
