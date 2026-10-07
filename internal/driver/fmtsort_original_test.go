package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestFmtsortOriginal(t *testing.T) {
	dir := filepath.Join(runtime.GOROOT(), "src/internal/fmtsort")
	out := t.TempDir()
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TrimPath: false})
	if err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	const filter = ""
	args := upstreamTestArgs("-test.run=" + filter)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			var result programResult
			if target == "vm" {
				result = upstreamOutcome(t, dir, nil, runLimit, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "fmtsort-whole-original", "internal/fmtsort", filter, target, dir, result.Text, result.Exit, false)
		})
	}
}
