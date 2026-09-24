package driver_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestDirectoryCursor(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("directory cursor validation requires macOS or Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/directorycursor\n\ngo 1.27\n")
	osDir := filepath.Join(runtime.GOROOT(), "src/os")
	writeFile(t, filepath.Join(dir, "os_test.go"), selectedUpstreamSource(t, filepath.Join(osDir, "os_test.go"), []string{"smallReaddirnames", "TestReaddirnamesOneAtATime", "TestReaddirNValues", "TestReaddirSmallSeek"}))
	if err := os.MkdirAll(filepath.Join(dir, "testdata", "issue37161"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a", "b", "c"} {
		data, err := os.ReadFile(filepath.Join(osDir, "testdata", "issue37161", name))
		if err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(dir, "testdata", "issue37161", name), string(data))
	}

	license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
	const filter = "^(TestReaddirnamesOneAtATime|TestReaddirNValues|TestReaddirSmallSeek)$"
	t.Log(run(t, dir, "go", "test", "-v", "-run", filter, "."))
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true})
	if err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			args := []string{"-test.v", "-test.run=" + filter}
			var result programResult
			if target == "vm" {
				result = programOutcome(t, dir, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "os-directory-cursor", "os", filter, target, osDir, result.Text, result.Exit)
		})
	}
}
