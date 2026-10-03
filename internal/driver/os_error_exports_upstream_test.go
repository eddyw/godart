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

func TestOSErrorExportsOriginal(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("Original error exports require a native filesystem")
	}
	dir := filepath.Join(runtime.GOROOT(), "src/os")
	files, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	sources := load.Filter{}
	for _, path := range files {
		switch filepath.Base(path) {
		case "os_test.go":
			sources[path] = []byte(selectedUpstreamSource(t, path, []string{"TestWriteAtInAppendMode"}))
		case "tempfile_test.go":
			sources[path] = []byte(selectedUpstreamSource(t, path, []string{"TestCreateTempBadPattern", "TestMkdirTempBadPattern"}))
		case "export_test.go":
			sources[path] = []byte(selectedUpstreamSource(t, path, []string{"ErrWriteAtInAppendMode", "ErrPatternHasSeparator"}))
		default:
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.PackageClauseOnly)
			if err != nil {
				t.Fatal(err)
			}
			sources[path] = []byte("package " + file.Name.Name + "\n")
		}
	}
	out := t.TempDir()
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TestSources: sources, TrimPath: true})
	if err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	const filter = "^(TestWriteAtInAppendMode|TestCreateTempBadPattern|TestMkdirTempBadPattern)$"
	args := upstreamTestArgs("-test.run=" + filter)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			var result programResult
			if target == "vm" {
				result = upstreamOutcome(t, dir, nil, runLimit, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "os-error-exports-original", "os", filter, target, dir, result.Text, result.Exit, true)
		})
	}
}
