package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestDirectoryCreationUmask(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("native directory modes require Linux or macOS")
	}
	source, err := filepath.Abs("../../testdata/programs/mkdirperm")
	if err != nil {
		t.Fatal(err)
	}
	native := filepath.Join(t.TempDir(), "program")
	run(t, source, "go", "build", "-o", native, ".")
	out := t.TempDir()
	if err := driver.Build(driver.Options{Dir: source, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Format: false}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	run(t, out, "dart", "compile", "exe", "bin/main.dart", "-o", "app")
	script := "set -e\numask \"$1\"\nshift\nexec \"$@\""
	for _, mask := range []string{"022", "027", "077"} {
		t.Run(mask, func(t *testing.T) {
			want := programOutcome(t, source, "sh", "-c", script, "directory-mode", mask, native)
			if want.Exit != 0 {
				t.Fatal(want.Text)
			}
			vm := programOutcome(t, out, "sh", "-c", script, "directory-mode", mask, "dart", "run", "--verbosity=error", "bin/main.dart")
			if vm != want {
				t.Errorf("VM directory modes differ: got %+v, want %+v", vm, want)
			}
			aot := programOutcome(t, out, "sh", "-c", script, "directory-mode", mask, filepath.Join(out, "app"))
			if aot != want {
				t.Errorf("AOT directory modes differ: got %+v, want %+v", aot, want)
			}
		})
	}
}
