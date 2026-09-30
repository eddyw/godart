package driver_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestCommandInitialStateUpstream(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("this adapter covers Unix command state")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/commandstate\n\ngo 1.27\n\nrequire example.com/testenv v0.0.0\nreplace example.com/testenv => ./testenv\n")
	source := selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/os/exec/exec_test.go"), []string{"TestString", "TestStringPathNotResolved", "TestNoPath", "TestStart_twice", "TestNoStringPanic"})
	source = strings.Replace(source, "package exec_test", "package commandstate_test", 1)
	source = strings.ReplaceAll(source, `"internal/testenv"`, `"example.com/testenv"`)
	writeFile(t, filepath.Join(dir, "command_test.go"), source)
	if err := os.Mkdir(filepath.Join(dir, "testenv"), 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "testenv/go.mod"), "module example.com/testenv\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "testenv/testenv.go"), `package testenv
import "testing"
func MustHaveExec(t testing.TB) {}
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
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			filter := "^Test(String|StringPathNotResolved|NoPath|Start_twice|NoStringPanic)$"
			if target == "js" || target == "wasm" {
				filter = "^Test(StringPathNotResolved|NoPath|NoStringPanic)$"
			}
			args := upstreamTestArgs("-test.run=" + filter)
			var got programResult
			if target == "vm" {
				got = upstreamOutcome(t, dir, nil, runLimit, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				got = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "exec-command-state", "os/exec", filter, target, filepath.Join(runtime.GOROOT(), "src/os/exec"), got.Text, got.Exit, true)
		})
	}
}
