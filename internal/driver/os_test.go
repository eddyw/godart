package driver_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
	want := goTest(t, dir, true)
	got := gartTest(t, dir, driver.TestOptions{Verbose: true})
	if got != want {
		t.Errorf("os environment output differs:\ngot:\n%s\nwant:\n%s", got, want)
	}
	for _, name := range []string{"TestExpand", "TestUnsetenv", "TestClearenv", "TestLookupEnv", "TestEnvironConsistency"} {
		if !strings.Contains(got, "--- PASS: "+name) {
			t.Errorf("missing result for %s", name)
		}
	}
}
