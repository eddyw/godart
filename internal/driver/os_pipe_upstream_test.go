package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestOSPipeOriginal(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("Pipe validation requires native descriptors")
	}
	dir := filepath.Join(runtime.GOROOT(), "src/os")
	fixture, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(fixture, "go.mod"), "module os_test\n\ngo 1.27\n")
	writeFile(t, filepath.Join(fixture, "000_package.go"), "package os_test\n")
	writeFile(t, filepath.Join(fixture, "os_test.go"), selectedUpstreamSource(t, filepath.Join(dir, "os_test.go"), []string{"TestFileIOCloseRace", "TestPipeIOCloseRace", "TestPipeCloseRace"}))
	writeFile(t, filepath.Join(fixture, "pipe_test.go"), selectedUpstreamSource(t, filepath.Join(dir, "pipe_test.go"), []string{"TestPipeEOF", "testPipeEOF", "TestEPIPE", "TestClosedPipeRaceRead", "TestClosedPipeRaceWrite", "testClosedPipeRace"}))
	stats, err := driver.BuildWithStats(driver.Options{Dir: fixture, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TrimPath: true})
	if err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	const filter = "^(TestFileIOCloseRace|TestPipeIOCloseRace|TestPipeCloseRace|TestPipeEOF|TestEPIPE|TestClosedPipeRaceRead|TestClosedPipeRaceWrite)$"
	args := upstreamTestArgs("-test.run=" + filter)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			var result programResult
			if target == "vm" {
				result = upstreamOutcome(t, dir, nil, runLimit, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "pipe-original", "os", filter, target, dir, result.Text, result.Exit, true)
		})
	}
}
