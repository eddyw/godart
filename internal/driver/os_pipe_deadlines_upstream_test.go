package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestOSPipeDeadlinesOriginal(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("Pipe deadline validation requires native descriptors")
	}
	dir := filepath.Join(runtime.GOROOT(), "src/os")
	fixture, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(fixture, "go.mod"), "module os_test\n\ngo 1.27\n")
	writeFile(t, filepath.Join(fixture, "000_package.go"), "package os_test\n")
	writeFile(t, filepath.Join(fixture, "timeout_test.go"), selectedUpstreamSource(t, filepath.Join(dir, "timeout_test.go"), []string{"TestReadTimeout", "TestReadTimeoutMustNotReturn", "TestWriteTimeout", "TestWriteTimeoutMustNotReturn", "TestReadTimeoutFluctuation", "TestWriteTimeoutFluctuation", "TestReadWriteDeadlineRace", "TestRacyRead", "TestRacyWrite", "TestNonpollableDeadline", "pipeDeadlineTest", "pipeDeadlinesTestCases", "noDeadline", "readTimeoutTests", "writeTimeoutTests", "minDynamicTimeout", "maxDynamicTimeout", "timeoutUpperBound", "nextTimeout"}))
	writeFile(t, filepath.Join(fixture, "timeout_unix_test.go"), selectedUpstreamSource(t, filepath.Join(dir, "timeout_unix_test.go"), []string{"init"}))
	writeFile(t, filepath.Join(fixture, "os_test.go"), selectedUpstreamSource(t, filepath.Join(dir, "os_test.go"), []string{"isDeadlineExceeded"}))
	stats, err := driver.BuildWithStats(driver.Options{Dir: fixture, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TrimPath: true})
	if err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	const filter = "^(TestReadTimeout|TestReadTimeoutMustNotReturn|TestWriteTimeout|TestWriteTimeoutMustNotReturn|TestReadTimeoutFluctuation|TestWriteTimeoutFluctuation|TestReadWriteDeadlineRace|TestRacyRead|TestRacyWrite|TestNonpollableDeadline)$"
	args := upstreamTestArgs("-test.run=" + filter)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			var result programResult
			if target == "vm" {
				result = upstreamOutcome(t, dir, nil, runLimit, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "pipe-deadlines-original", "os", filter, target, dir, result.Text, result.Exit, true)
		})
	}
}
