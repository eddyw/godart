package driver_test

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/load"
)

func TestBigIntText(t *testing.T) {
	const pkg = "math/big"
	dir := filepath.Join(runtime.GOROOT(), "src", pkg)
	files, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	sources := load.Filter{}
	for _, path := range files {
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.PackageClauseOnly)
		if err != nil {
			t.Fatal(err)
		}
		sources[path] = []byte("package " + file.Name.Name + "\n")
	}
	path := filepath.Join(dir, "intconv_test.go")
	source := selectedUpstreamSource(t, path, []string{"TestIntText", "TestAppendText", "TestGetString", "TestSetString", "TestFormat", "TestScan", "stringTests", "format", "formatTests", "testFormat", "scanTest", "scanTests", "init"})
	// The original versioned import has the package name rand.
	source = strings.Replace(source, "import (\n", "import (\nrand \"math/rand/v2\"\n", 1)
	sources[path] = []byte(source)
	path = filepath.Join(dir, "int_test.go")
	sources[path] = []byte(selectedUpstreamSource(t, path, []string{"isNormalized"}))
	out := t.TempDir()
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TestSources: sources, TrimPath: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.Binaries) != 1 {
		t.Fatal(stats.Binaries)
	}
	prepareDart(t, out)
	const filter = "^(TestIntText|TestAppendText|TestGetString|TestSetString|TestFormat|TestScan)$"
	args := upstreamTestArgs("-test.run=" + filter)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			var result programResult
			if target == "vm" {
				result = upstreamOutcome(t, dir, nil, upstreamLimit(), "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "big-int-text", pkg, filter, target, dir, result.Text, result.Exit, true)
		})
	}
}
