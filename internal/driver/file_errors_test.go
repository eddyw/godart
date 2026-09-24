package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFileErrorsAOT(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("file errors validation requires macOS or Linux")
	}
	dir, err := filepath.Abs("../../testdata/programs/fileerrors")
	if err != nil {
		t.Fatal(err)
	}
	want := runProgram(t, dir, "go", "run", ".")
	out := t.TempDir()
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	exe := filepath.Join(out, "program")
	run(t, out, "dart", "compile", "exe", "bin/main.dart", "-o", exe)
	if got := runProgram(t, out, exe); got != want {
		t.Fatalf("file errors differs\nGo:\n%s\nGart:\n%s", want, got)
	}
}
