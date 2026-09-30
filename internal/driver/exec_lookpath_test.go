package driver_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestExecutableLookPathUpstream(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("this adapter covers Unix executable lookup")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/lookpath\n\ngo 1.27\n\nrequire example.com/testenv v0.0.0\nreplace example.com/testenv => ./testenv\n")
	source := selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/os/exec/dot_test.go"), []string{"pathVar", "TestLookPath"})
	source = strings.Replace(source, "package exec_test", "package lookpath_test", 1)
	source = strings.ReplaceAll(source, `"internal/testenv"`, `"example.com/testenv"`)
	writeFile(t, filepath.Join(dir, "lookpath_test.go"), source)
	if err := os.Mkdir(filepath.Join(dir, "testenv"), 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "testenv/go.mod"), "module example.com/testenv\n\ngo 1.27\n")
	// This test uses Executable only as an existing executable file in PATH.
	writeFile(t, filepath.Join(dir, "testenv/testenv.go"), `package testenv
import("os";"testing")
func MustHaveExec(t testing.TB) {}
func Executable(t testing.TB) string {t.Helper();path,err:=os.Executable();if err!=nil{t.Fatal(err)};return path}
`)
	license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
	t.Log(run(t, dir, "go", "test", "-v", "."))
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true})
	if err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			args := upstreamTestArgs("-test.run=^TestLookPath$")
			var got programResult
			if target == "vm" {
				got = upstreamOutcome(t, dir, nil, runLimit, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				got = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "exec-lookpath", "os/exec", "^TestLookPath$", target, filepath.Join(runtime.GOROOT(), "src/os/exec"), got.Text, got.Exit, true)
		})
	}
}
