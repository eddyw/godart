package driver_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestTestingHostContracts(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	// TestChdir requires a current directory without a symlink alias.
	dir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/testinghost\n\ngo 1.27\n")
	source := selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/testing/testing_test.go"), []string{
		"TestTempDirInCleanup", "TestTempDirInBenchmark", "TestTempDir", "testTempDir", "TestTempDirGOTMPDIR", "TestSetenv", "TestChdir", "TestContext",
	})
	writeFile(t, filepath.Join(dir, "upstream_test.go"), source)
	cleanup := selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/testing/sub_test.go"), []string{"TestCleanup", "TestConcurrentCleanup", "TestCleanupCalledEvenAfterGoexit", "TestRunCleanup", "TestCleanupParallelSubtests", "TestNestedCleanup"})
	cleanup = strings.Replace(cleanup, "package testing\n", "package testing_test\nimport . \"testing\"\n", 1)
	writeFile(t, filepath.Join(dir, "cleanup_test.go"), cleanup)

	license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true})
	if err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	const filter = "^Test(TempDirInCleanup|TempDirInBenchmark|TempDir|TempDirGOTMPDIR|Setenv|Chdir|Context|Cleanup|ConcurrentCleanup|CleanupCalledEvenAfterGoexit|RunCleanup|CleanupParallelSubtests|NestedCleanup)$"
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			args := []string{"-test.v", "-test.short=" + strconv.FormatBool(testing.Short()), "-test.run=" + filter}
			var result programResult
			if target == "vm" {
				result = programOutcome(t, dir, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "testing-host", "testing", filter, target, filepath.Join(runtime.GOROOT(), "src/testing"), result.Text, result.Exit)
		})
	}
}
