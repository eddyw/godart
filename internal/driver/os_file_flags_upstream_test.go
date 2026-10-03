package driver_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestOSFileFlagsOriginal(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("File flag validation requires a native filesystem")
	}
	dir := filepath.Join(runtime.GOROOT(), "src/os")
	fixture, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(fixture, "go.mod"), "module os_test\n\ngo 1.27\n\nrequire example.com/testenv v0.0.0\nreplace example.com/testenv => ./testenv\n")
	writeFile(t, filepath.Join(fixture, "000_package.go"), "package os_test\n")
	source := selectedUpstreamSource(t, filepath.Join(dir, "os_test.go"), []string{"TestOpenNoName", "TestFileRDWRFlags", "TestStatDirWithTrailingSlash", "TestOpenFileCreateExclDanglingSymlink"})
	source = strings.ReplaceAll(source, `"internal/testenv"`, `"example.com/testenv"`)
	writeFile(t, filepath.Join(fixture, "os_test.go"), source)
	writeFile(t, filepath.Join(fixture, "root_test.go"), selectedUpstreamSource(t, filepath.Join(dir, "root_test.go"), []string{"testMaybeRooted"}))
	if err := os.Mkdir(filepath.Join(fixture, "testenv"), 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(fixture, "testenv/go.mod"), "module example.com/testenv\n\ngo 1.27\n")
	testenv := filepath.Join(runtime.GOROOT(), "src/internal/testenv")
	for _, file := range []struct {
		name         string
		declarations []string
	}{
		{"testenv.go", []string{"MustHaveSymlink", "SyscallIsNotSupported"}},
		{"testenv_notwin.go", []string{"hasSymlink"}},
		{"testenv_unix.go", []string{"syscallIsNotSupported"}},
	} {
		writeFile(t, filepath.Join(fixture, "testenv", file.name), selectedUpstreamSource(t, filepath.Join(testenv, file.name), file.declarations))
	}
	stats, err := driver.BuildWithStats(driver.Options{Dir: fixture, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TrimPath: true})
	if err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	const filter = "^(TestOpenNoName|TestFileRDWRFlags|TestStatDirWithTrailingSlash|TestOpenFileCreateExclDanglingSymlink)$"
	args := upstreamTestArgs("-test.run=" + filter)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			var result programResult
			if target == "vm" {
				result = upstreamOutcome(t, dir, nil, runLimit, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "file-flags-original", "os", filter, target, dir, result.Text, result.Exit, true)
		})
	}
}
