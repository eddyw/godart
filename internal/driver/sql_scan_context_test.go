package driver_test

import (
	"path/filepath"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestSQLScanContext(t *testing.T) {
	dir, err := filepath.Abs("../../testdata/programs/sqlmem")
	if err != nil {
		t.Fatal(err)
	}
	want := runProgram(t, dir, "go", "run", ".")
	out := t.TempDir()
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	if got := runProgram(t, dir, "dart", filepath.Join(out, "bin/main.dart")); got != want {
		t.Fatalf("Go:\n%s\nGart:\n%s", want, got)
	}
	for _, target := range []string{"aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			got := runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
			if got.Exit != 0 || got.Text != want {
				t.Fatalf("%s: exit %d\nGo:\n%s\nGart:\n%s", target, got.Exit, want, got.Text)
			}
		})
	}
}
