package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"testing"
)

func TestTestingChdirRestore(t *testing.T) {
	dir, err := filepath.Abs("../../testdata/tests/chdirrename")
	if err != nil {
		t.Fatal(err)
	}
	native := filepath.Join(t.TempDir(), "test")
	run(t, dir, "go", "test", "-c", "-o", native, ".")
	nativeResult := programOutcome(t, dir, native, "-test.v")
	if nativeResult.Exit != 0 {
		t.Fatal(nativeResult.Text)
	}
	want := normalize(nativeResult.Text)
	out := t.TempDir()
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true})
	if err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			args := []string{"-test.v"}
			var result programResult
			if target == "vm" {
				result = programOutcome(t, dir, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			if result.Exit != 0 || normalize(result.Text) != want {
				t.Fatalf("got exit %d:\n%s\nwant:\n%s", result.Exit, result.Text, want)
			}
		})
	}
}
