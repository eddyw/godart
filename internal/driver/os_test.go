package driver_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

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
