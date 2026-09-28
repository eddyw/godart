package driver_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/eddyw/godart/internal/driver"
)

func TestFileStatPrecision(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file")
	writeFile(t, path, "metadata")
	stamp := time.Unix(1700000000, 123456789)
	if err := os.Chtimes(path, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	source, err := filepath.Abs("../../testdata/programs/filestat")
	if err != nil {
		t.Fatal(err)
	}
	runtimeDir, err := filepath.Abs("../../runtime/goruntime")
	if err != nil {
		t.Fatal(err)
	}
	native := filepath.Join(t.TempDir(), "program")
	run(t, source, "go", "build", "-o", native, ".")
	want := programOutcome(t, source, native, path)
	out := t.TempDir()
	if err := driver.Build(driver.Options{Dir: source, Patterns: []string{"."}, Out: out, Runtime: runtimeDir, Format: false}); err != nil {
		t.Fatal(err)
	}
	run(t, out, "dart", "pub", "get")
	if got := programOutcome(t, out, "dart", "bin/main.dart", path); got != want {
		t.Errorf("VM file metadata differs: got %+v, want %+v", got, want)
	}
	run(t, out, "dart", "compile", "exe", "bin/main.dart", "-o", "app")
	if got := programOutcome(t, out, filepath.Join(out, "app"), path); got != want {
		t.Errorf("AOT file metadata differs: got %+v, want %+v", got, want)
	}
}

// The complete os suite requires native descriptors and test-only exports.
// Its environment tests use the public API without those dependencies.
func TestOSEnvironment(t *testing.T) {
	dir := t.TempDir()
	source, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "src/os/env_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/oscompat\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "env_test.go"), string(source))
	license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
	out := t.TempDir()
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.Binaries) != 1 {
		t.Fatalf("test binaries: %v", stats.Binaries)
	}
	prepareDart(t, out)
	const filter = "^(TestExpand|TestConsistentEnviron|TestUnsetenv|TestClearenv|TestLookupEnv|TestEnvironConsistency)$"
	args := upstreamTestArgs("-test.run=" + filter)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			var got programResult
			if target == "vm" {
				got = programOutcome(t, dir, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				got = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "os-environment", "os", filter, target, filepath.Join(runtime.GOROOT(), "src/os"), got.Text, got.Exit, false)
		})
	}
}
