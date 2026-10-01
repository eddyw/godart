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

func TestBigIntArithmetic(t *testing.T) {
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
	sources[path] = []byte(selectedUpstreamSource(t, path, []string{"TestBinomial", "TestBitLen", "TestBitSet", "TestBitwise", "TestCmpAbs", "TestFloat64", "TestGcd", "TestIntCmpSelf", "TestIntDivide", "TestIssue22830", "TestIssue2607", "TestJacobi", "TestJacobiPanic", "TestLsh", "TestLshRsh", "TestLshSelf", "TestModInverse", "TestMul", "TestMulRangeZ", "TestNot", "TestQuo", "TestQuoStepD6", "TestRsh", "TestRshSelf", "TestSqrt", "TestTrailingZeroBits", "altBit", "altSetBit", "bitFun", "bitLenTests", "bitsetTests", "bitwiseTests", "checkGcd", "checkLehmerExtGcd", "checkLehmerGcd", "checkMul", "checkQuo", "cmpAbsTests", "euclidExtGCD", "gcdTests", "intShiftTest", "isNormalized", "lshTests", "modInverseTests", "mulBytes", "mulRangesZ", "notTests", "quoTests", "rshTests", "testBitFun", "testBitFunSelf", "testBitset", "testGcd", "tzbTests"}))
	path = filepath.Join(dir, "nat_test.go")
	sources[path] = []byte(selectedUpstreamSource(t, path, []string{"mulRangesN"}))
	out := t.TempDir()
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TestSources: sources, TrimPath: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.Binaries) != 1 {
		t.Fatal(stats.Binaries)
	}
	prepareDart(t, out)
	const filter = "^(TestMul|TestMulRangeZ|TestBinomial|TestQuo|TestQuoStepD6|TestIntDivide|TestBitLen|TestGcd|TestRsh|TestRshSelf|TestLsh|TestLshSelf|TestLshRsh|TestCmpAbs|TestIntCmpSelf|TestBitSet|TestTrailingZeroBits|TestBitwise|TestNot|TestModInverse|TestJacobi|TestJacobiPanic|TestIssue2607|TestSqrt|TestIssue22830|TestFloat64)$"
	args := upstreamTestArgs("-test.run=" + filter)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			var result programResult
			if target == "vm" {
				result = upstreamOutcome(t, dir, nil, upstreamLimit(), "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "big-int-arithmetic", pkg, filter, target, dir, result.Text, result.Exit, true)
		})
	}
}
