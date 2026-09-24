package driver_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestWriteFilePermissions(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("file permission validation requires macOS or Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/writefile\n\ngo 1.27\n")
	osDir := filepath.Join(runtime.GOROOT(), "src/os")
	for _, item := range []upstreamFile{
		{Name: "os_test.go", Declarations: []string{"TestFilePermissions", "TestAppendDoesntOverwrite"}},
		{Name: "read_test.go", Declarations: []string{"TestWriteFile", "TestReadOnlyWriteFile"}},
		{Name: "root_test.go", Declarations: []string{"testMaybeRooted"}},
	} {
		writeFile(t, filepath.Join(dir, item.Name), selectedUpstreamSource(t, filepath.Join(osDir, item.Name), item.Declarations))
	}
	license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
	const filter = "^(TestFilePermissions|TestAppendDoesntOverwrite|TestWriteFile|TestReadOnlyWriteFile)$"
	t.Log(run(t, dir, "go", "test", "-short="+strconv.FormatBool(testing.Short()), "-v", "-run", filter, "."))
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true})
	if err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			args := []string{"-test.v", "-test.short=" + strconv.FormatBool(testing.Short()), "-test.run=" + filter}
			var result programResult
			if target == "vm" {
				result = programOutcome(t, dir, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "os-writefile-permissions", "os", filter, target, osDir, result.Text, result.Exit)
		})
	}
}
