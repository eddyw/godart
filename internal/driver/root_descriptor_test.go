package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRootDescriptorUpstream(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("root filesystem methods require a native filesystem")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/rootdescriptor\n\ngo 1.27\n")
	names := []string{"TestRootRenameAfterOpen", "TestRootRaceRenameDir"}
	pattern := "^(TestRootRenameAfterOpen|TestRootRaceRenameDir)$"
	source := selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/os/root_test.go"), names)
	writeFile(t, filepath.Join(dir, "rootdescriptor_test.go"), source)
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
	targets := []string{"vm", "aot"}
	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			targetPattern := pattern
			args := upstreamTestArgs("-test.run=" + targetPattern)
			var got programResult
			if target == "vm" {
				got = upstreamOutcome(t, dir, nil, runLimit, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				got = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "root-descriptor", "os", targetPattern, target, filepath.Join(runtime.GOROOT(), "src/os"), got.Text, got.Exit, true)
		})
	}
}
