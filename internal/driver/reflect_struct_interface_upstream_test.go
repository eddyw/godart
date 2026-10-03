package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestReflectStructInterfaceOriginal(t *testing.T) {
	dir := filepath.Join(runtime.GOROOT(), "src/reflect")
	fixture, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(fixture, "go.mod"), "module reflect_test\n\ngo 1.27\n")
	writeFile(t, filepath.Join(fixture, "000_package.go"), "package reflect_test\n")
	writeFile(t, filepath.Join(fixture, "type_test.go"), selectedUpstreamSource(t, filepath.Join(dir, "type_test.go"), []string{"TestStructOfEmbeddedIfaceMethodCall"}))
	writeFile(t, filepath.Join(fixture, "all_test.go"), selectedUpstreamSource(t, filepath.Join(dir, "all_test.go"), []string{"shouldPanic"}))
	stats, err := driver.BuildWithStats(driver.Options{Dir: fixture, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TrimPath: true})
	if err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	const filter = "^TestStructOfEmbeddedIfaceMethodCall$"
	args := upstreamTestArgs("-test.run=" + filter)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			var result programResult
			if target == "vm" {
				result = upstreamOutcome(t, dir, nil, runLimit, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "reflect-struct-interface-original", "reflect", filter, target, dir, result.Text, result.Exit, true)
		})
	}
}
