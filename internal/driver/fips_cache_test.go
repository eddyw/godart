package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestFIPSCache(t *testing.T) {
	const pkg = "crypto/internal/fips140cache"
	dir, out := filepath.Join(runtime.GOROOT(), "src", pkg), t.TempDir()
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TrimPath: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.Binaries) != 1 {
		t.Fatal(stats.Binaries)
	}
	prepareDart(t, out)
	prelude := filepath.Join(out, "collect.mjs")
	writeFile(t, prelude, "const timer = setInterval(() => global.gc(), 10); timer.unref();\n")
	args := upstreamTestArgs("-test.run=^TestCache$")
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			var result programResult
			if target == "vm" {
				result = upstreamOutcome(t, dir, nil, upstreamLimit(), "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args, "--expose-gc", "--import", prelude)
			}
			compareUpstream(t, "fips-cache", pkg, "^TestCache$", target, dir, result.Text, result.Exit, true)
		})
	}
}
