package driver_test

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/load"
)

func TestSysrandRead(t *testing.T) {
	heap := "--max-old-space-size=4096"
	if !testing.Short() {
		// Full compression of the original 40 MiB buffer exceeds the 4 GiB JS heap.
		heap = "--max-old-space-size=8192"
	}
	testSysrand(t, []string{"TestRead", "TestReadByteValues", "TestReadEmpty"}, heap)
}

func TestSysrandConcurrentRead(t *testing.T) {
	if testing.Short() {
		t.Skip("the original concurrent-read assertion requires full mode")
	}
	testSysrand(t, []string{"TestConcurrentRead"})
}

func testSysrand(t *testing.T, tests []string, nodeArgs ...string) {
	t.Helper()
	const pkg = "crypto/internal/sysrand"
	dir := filepath.Join(runtime.GOROOT(), "src", pkg)
	path := filepath.Join(dir, "rand_test.go")
	sources := load.Filter{path: []byte(selectedUpstreamSource(t, path, tests))}
	sources[filepath.Join(dir, "rand_linux_test.go")] = []byte("package sysrand_test\n")
	out := t.TempDir()
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TestSources: sources, TrimPath: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.Binaries) != 1 {
		t.Fatal(stats.Binaries)
	}
	prepareDart(t, out)
	filter := "^(" + strings.Join(tests, "|") + ")$"
	args := upstreamTestArgs("-test.run=" + filter)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			var result programResult
			if target == "vm" {
				result = upstreamOutcome(t, dir, nil, upstreamLimit(), "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args, nodeArgs...)
			}
			compareUpstream(t, "sysrand-portable", pkg, filter, target, dir, result.Text, result.Exit, true)
		})
	}
}
