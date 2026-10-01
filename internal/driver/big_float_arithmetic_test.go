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

func TestBigFloatArithmetic(t *testing.T) {
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
		sources[path] = []byte(selectedUpstreamSource(t, path, []string{"Bits"}))
	}
	{
		path := filepath.Join(dir, "float_test.go")
		sources[path] = []byte(selectedUpstreamSource(t, path, []string{"Float.uint64", "Float.int64", "TestFloatAbs", "TestFloatAdd", "TestFloatAdd32", "TestFloatAdd64", "TestFloatAddRoundZero", "TestFloatArithmeticOverflow", "TestFloatArithmeticRounding", "TestFloatArithmeticSpecialValues", "TestFloatCmpSpecialValues", "TestFloatFloat32", "TestFloatFloat64", "TestFloatInc", "TestFloatInt", "TestFloatInt64", "TestFloatIsInt", "TestFloatMantExp", "TestFloatMantExpAliasing", "TestFloatMinPrec", "TestFloatMul", "TestFloatMul64", "TestFloatNeg", "TestFloatPredicates", "TestFloatQuo", "TestFloatQuoSmoke", "TestFloatRat", "TestFloatRound", "TestFloatRound24", "TestFloatSetFloat64", "TestFloatSetInf", "TestFloatSetInt", "TestFloatSetInt64", "TestFloatSetMantExp", "TestFloatSetPrec", "TestFloatSetRat", "TestFloatSetUint64", "TestFloatSign", "TestFloatUint64", "TestFloatZeroValue", "TestIssue20490", "TestIssue6866", "alike", "alike32", "alike64", "bitsList", "fromBinary", "long", "makeFloat", "precList", "testFloatRound", "toBinary"}))
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
	const filter = "^(TestFloatZeroValue|TestFloatSetPrec|TestFloatMinPrec|TestFloatSign|TestFloatMantExp|TestFloatMantExpAliasing|TestFloatSetMantExp|TestFloatPredicates|TestFloatIsInt|TestFloatRound|TestFloatRound24|TestFloatSetUint64|TestFloatSetInt64|TestFloatSetFloat64|TestFloatSetInt|TestFloatSetRat|TestFloatSetInf|TestFloatUint64|TestFloatInt64|TestFloatFloat32|TestFloatFloat64|TestFloatInt|TestFloatRat|TestFloatAbs|TestFloatNeg|TestFloatInc|TestFloatAdd|TestFloatAddRoundZero|TestFloatAdd32|TestFloatAdd64|TestIssue20490|TestFloatMul|TestFloatMul64|TestIssue6866|TestFloatQuo|TestFloatQuoSmoke|TestFloatArithmeticSpecialValues|TestFloatArithmeticOverflow|TestFloatArithmeticRounding|TestFloatCmpSpecialValues)$"
	args := upstreamTestArgs("-test.run=" + filter)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			var result programResult
			if target == "vm" {
				result = upstreamOutcome(t, dir, nil, upstreamLimit(), "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "TestBigFloatArithmetic", pkg, filter, target, dir, result.Text, result.Exit, true)
		})
	}
}
