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

func TestBigRationalPortable(t *testing.T) {
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
		path := filepath.Join(dir, "float_test.go")
		sources[path] = []byte(selectedUpstreamSource(t, path, []string{"long"}))
	}
	{
		path := filepath.Join(dir, "intmarsh_test.go")
		sources[path] = []byte(selectedUpstreamSource(t, path, []string{"TestGobEncodingNilIntInSlice", "TestIntAppendText", "TestIntAppendTextNil", "TestIntGobEncoding", "TestIntJSONEncoding", "TestIntJSONEncodingNil", "TestIntXMLEncoding", "encodingTests"}))
	}
	{
		path := filepath.Join(dir, "rat_test.go")
		sources[path] = []byte(selectedUpstreamSource(t, path, []string{"TestDenomRace", "TestFloat32Distribution", "TestFloat64Distribution", "TestIsFinite", "TestIsInt", "TestIssue2379", "TestIssue34919", "TestIssue3521", "TestIssue820", "TestRatAbs", "TestRatBin", "TestRatCmp", "TestRatInv", "TestRatNeg", "TestRatSetFrac64Rat", "TestRatSetInt64", "TestRatSetUint64", "TestRatSign", "TestSetFloat64NonFinite", "TestZeroRat", "checkIsBestApprox32", "checkIsBestApprox64", "checkNonLossyRoundtrip32", "checkNonLossyRoundtrip64", "delta", "isEven32", "isEven64", "ratBinArg", "ratBinFun", "ratBinTests", "ratCmpTests", "setFrac64Tests", "testRatBin"}))
	}
	{
		path := filepath.Join(dir, "ratconv_test.go")
		sources[path] = []byte(selectedUpstreamSource(t, path, []string{"StringTest", "TestFloat32SpecialCases", "TestFloat64SpecialCases", "TestFloatPrec", "TestFloatString", "TestIssue31184", "TestRatScan", "TestRatSetString", "TestRatSetStringZero", "TestScanExponent", "exponentTests", "float64inputs", "floatStringTests", "isFinite", "setStringTests", "setStringTests2"}))
	}
	{
		path := filepath.Join(dir, "ratmarsh_test.go")
		sources[path] = []byte(selectedUpstreamSource(t, path, []string{"TestGobEncodingNilRatInSlice", "TestRatAppendText", "TestRatGobDecodeShortBuffer", "TestRatGobEncoding", "TestRatJSONEncoding", "TestRatXMLEncoding", "ratDenoms", "ratNums"}))
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
	const filter = "^(TestZeroRat|TestRatSign|TestRatCmp|TestIsInt|TestRatAbs|TestRatNeg|TestRatInv|TestRatBin|TestIssue820|TestRatSetFrac64Rat|TestIssue2379|TestIssue3521|TestFloat32Distribution|TestFloat64Distribution|TestSetFloat64NonFinite|TestIsFinite|TestRatSetInt64|TestRatSetUint64|TestIssue34919|TestDenomRace|TestScanExponent|TestRatSetString|TestRatSetStringZero|TestRatScan|TestFloatString|TestFloat32SpecialCases|TestFloat64SpecialCases|TestIssue31184|TestFloatPrec|TestRatGobEncoding|TestGobEncodingNilRatInSlice|TestRatJSONEncoding|TestRatXMLEncoding|TestRatGobDecodeShortBuffer|TestRatAppendText|TestIntGobEncoding|TestGobEncodingNilIntInSlice|TestIntJSONEncoding|TestIntJSONEncodingNil|TestIntXMLEncoding|TestIntAppendText|TestIntAppendTextNil)$"
	args := upstreamTestArgs("-test.run=" + filter)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			var result programResult
			if target == "vm" {
				result = upstreamOutcome(t, dir, nil, upstreamLimit(), "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "big-rational-portable", pkg, filter, target, dir, result.Text, result.Exit, true)
		})
	}
}
