package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFmtOriginalPrinting(t *testing.T) {
	const pkg = "fmt"
	dir := filepath.Join(runtime.GOROOT(), "src", pkg)
	out := t.TempDir()
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TrimPath: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.Binaries) != 1 {
		t.Fatal(stats.Binaries)
	}
	prepareDart(t, out)
	const filter = "^(TestSprintf|TestFmtInterface|TestStringer|TestFormatString)$"
	args := upstreamTestArgs("-test.run=" + filter)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			var result programResult
			if target == "vm" {
				result = upstreamOutcome(t, dir, nil, upstreamLimit(), "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "TestFmtOriginalPrinting", pkg, filter, target, dir, result.Text, result.Exit, true)
		})
	}
}
