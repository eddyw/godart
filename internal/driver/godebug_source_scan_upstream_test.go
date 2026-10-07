package driver_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestGODEBUGSourceScanOriginal(t *testing.T) {
	dir := filepath.Join(runtime.GOROOT(), "src/internal/godebugs")
	out := t.TempDir()
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TrimPath: true})
	if err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	const filter = "^(TestAll|TestRemoved)$"
	args := upstreamTestArgs("-test.run=" + filter)
	env := append(os.Environ(), "GOROOT="+runtime.GOROOT())
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			var result programResult
			if target == "vm" {
				result = upstreamOutcome(t, dir, env, upstreamLimit(), "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTargetEnv(t, target, out, stats.Binaries[0].Entry, dir, args, env)
			}
			compareUpstream(t, "godebug-source-scan-original", "internal/godebugs", filter, target, dir, result.Text, result.Exit, true)
		})
	}
}
