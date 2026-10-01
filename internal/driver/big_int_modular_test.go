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

func TestBigIntModular(t *testing.T) {
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
	path := filepath.Join(dir, "int_test.go")
	sources[path] = []byte(selectedUpstreamSource(t, path, []string{"TestExp", "TestModSqrt", "expTests", "testModSqrt", "isNormalized"}))
	path = filepath.Join(dir, "prime_test.go")
	sources[path] = []byte(selectedUpstreamSource(t, path, []string{"primes"}))
	out := t.TempDir()
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TestSources: sources, TrimPath: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.Binaries) != 1 {
		t.Fatal(stats.Binaries)
	}
	prepareDart(t, out)
	const filter = "^(TestExp|TestModSqrt)$"
	args := upstreamTestArgs("-test.run=" + filter)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			var result programResult
			if target == "vm" {
				result = upstreamOutcome(t, dir, nil, upstreamLimit(), "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "big-int-modular", pkg, filter, target, dir, result.Text, result.Exit, true)
		})
	}
}
