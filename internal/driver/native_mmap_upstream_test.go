package driver_test

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/load"
)

func TestNativeMemoryMapOriginal(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("Original memory map assertions require native platform types")
	}
	dir := filepath.Join(runtime.GOROOT(), "src/syscall")
	files, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	sources := load.Filter{}
	for _, path := range files {
		switch filepath.Base(path) {
		case "mmap_unix_test.go":
			sources[path] = []byte(selectedUpstreamSource(t, path, []string{"TestMmap"}))
		default:
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.PackageClauseOnly)
			if err != nil {
				t.Fatal(err)
			}
			sources[path] = []byte("package " + file.Name.Name + "\n")
		}
	}
	native := upstreamOutcome(t, dir, nil, runLimit, "go", "test", "-v", "-run", "^TestMmap$", "-count=1")
	if native.Exit != 0 {
		t.Fatalf("Native original: exit %d\n%s", native.Exit, native.Text)
	}
	t.Log(native.Text)
	out := t.TempDir()
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TestSources: sources, TrimPath: true})
	if err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	const filter = "^TestMmap$"
	args := upstreamTestArgs("-test.run=" + filter)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			var result programResult
			if target == "vm" {
				result = upstreamOutcome(t, dir, nil, runLimit, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "native-memory-map-original", "syscall", filter, target, dir, result.Text, result.Exit, true)
		})
	}
}
