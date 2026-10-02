package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestRuntimePanicNil(t *testing.T) {
	dir := filepath.Join(runtime.GOROOT(), "src/runtime")
	fixture, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(fixture, "go.mod"), "module runtime_test\n\ngo 1.27\n")
	writeFile(t, filepath.Join(fixture, "000_package.go"), "package runtime_test\n")
	source := selectedUpstreamSource(t, filepath.Join(dir, "panic_test.go"), []string{"TestPanicNilErrorPrefix"})
	writeFile(t, filepath.Join(fixture, "panic_test.go"), source)

	stats, err := driver.BuildWithStats(driver.Options{Dir: fixture, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TrimPath: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.Binaries) != 1 {
		t.Fatalf("got %d binaries", len(stats.Binaries))
	}
	prepareDart(t, out)
	const filter = "^TestPanicNilErrorPrefix$"
	args := upstreamTestArgs("-test.run=" + filter)
	result := upstreamOutcome(t, dir, nil, runLimit, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
	compareUpstream(t, "runtime-panic-nil", "runtime", filter, "vm", dir, result.Text, result.Exit, true)
	for _, target := range []string{"aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			result := runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			compareUpstream(t, "runtime-panic-nil", "runtime", filter, target, dir, result.Text, result.Exit, true)
		})
	}
}
