package driver

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// The output package depends on the runtime by path. A new dependency of the
// runtime thus reaches the configuration of the output, and the package file
// of the output does not change with it.
func TestFreshReadsThePackageFileOfTheRuntime(t *testing.T) {
	dir := t.TempDir()
	config := filepath.Join(dir, "package_config.json")
	out := filepath.Join(dir, "out.yaml")
	runtime := filepath.Join(dir, "runtime.yaml")
	for _, name := range []string{out, runtime, config} {
		if err := os.WriteFile(name, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	old := time.Now().Add(-time.Hour)
	for _, name := range []string{out, runtime} {
		if err := os.Chtimes(name, old, old); err != nil {
			t.Fatal(err)
		}
	}
	if !fresh(config, []string{out, runtime}) {
		t.Fatal("a configuration above each package file reads as stale")
	}
	later := time.Now().Add(time.Hour)
	if err := os.Chtimes(runtime, later, later); err != nil {
		t.Fatal(err)
	}
	if fresh(config, []string{out, runtime}) {
		t.Fatal("a newer package file of the runtime reads as fresh")
	}
}
