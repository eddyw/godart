package driver_test

import (
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestRuntimeCaller(t *testing.T) {
	dir := filepath.Join(runtime.GOROOT(), "src/runtime")
	fixture, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(fixture, "go.mod"), "module example.com/upstream\n\ngo 1.27\n")
	source := selectedUpstreamSource(t, filepath.Join(dir, "symtab_test.go"), []string{"TestCaller", "testCallerFoo", "testCallerBar"})
	writeFile(t, filepath.Join(fixture, "symtab_test.go"), source)
	stats, err := driver.BuildWithStats(driver.Options{Dir: fixture, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TrimPath: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.Binaries) != 1 {
		t.Fatalf("got %d binaries", len(stats.Binaries))
	}
	prepareDart(t, out)
	const filter = "^TestCaller$"
	args := []string{"-test.v", "-test.short=" + strconv.FormatBool(testing.Short()), "-test.run=" + filter}
	result := programOutcome(t, dir, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
	compareUpstream(t, "runtime-caller", "runtime", filter, "vm", dir, result.Text, result.Exit)
	t.Run("aot", func(t *testing.T) {
		result := runUpstreamTarget(t, "aot", out, stats.Binaries[0].Entry, dir, args)
		compareUpstream(t, "runtime-caller", "runtime", filter, "aot", dir, result.Text, result.Exit)
	})
}
