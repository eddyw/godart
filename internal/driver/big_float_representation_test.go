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

func TestBigFloatRepresentation(t *testing.T) {
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
	{
		path := filepath.Join(dir, "bits_test.go")
		sources[path] = []byte(selectedUpstreamSource(t, path, []string{"Bits", "TestMulBits", "TestNormBits", "TestFromBits"}))
	}
	{
		path := filepath.Join(dir, "decimal_test.go")
		sources[path] = []byte(selectedUpstreamSource(t, path, []string{"TestDecimalString", "TestDecimalInit", "TestDecimalRounding"}))
	}
	{
		path := filepath.Join(dir, "sqrt_test.go")
		sources[path] = []byte(selectedUpstreamSource(t, path, []string{"TestFloatSqrt64", "TestFloatSqrt", "TestFloatSqrtSpecial"}))
	}

	out := t.TempDir()
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TestSources: sources, TrimPath: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.Binaries) != 1 {
		t.Fatal(stats.Binaries)
	}
	prepareDart(t, out)
	const filter = "^(TestMulBits|TestNormBits|TestFromBits|TestDecimalString|TestDecimalInit|TestDecimalRounding|TestFloatSqrt64|TestFloatSqrt|TestFloatSqrtSpecial)$"
	args := upstreamTestArgs("-test.run=" + filter)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			var result programResult
			if target == "vm" {
				result = upstreamOutcome(t, dir, nil, upstreamLimit(), "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "big-float-representation", pkg, filter, target, dir, result.Text, result.Exit, true)
		})
	}
}
