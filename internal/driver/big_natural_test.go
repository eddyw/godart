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

func TestBigNatural(t *testing.T) {
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
		path := filepath.Join(dir, "nat_test.go")
		source := selectedUpstreamSource(t, path, []string{"TestAdd", "TestBit", "TestCmp", "TestExpNN", "TestFibo", "TestIssue37499", "TestIssue42552", "TestModW", "TestMontgomery", "TestMulRangeN", "TestNLZ", "TestNatMul", "TestNatSqr", "TestNatSubMod2N", "TestSet", "TestShiftLeft", "TestShiftRight", "TestSticky", "TestSub", "argNN", "bitTests", "cmpTests", "expNNTests", "fibo", "fiboNums", "funNN", "funSNN", "leftShiftTests", "modWTest", "modWTests32", "modWTests64", "montgomeryTests", "mulRangesN", "natFromString", "permute", "prodNN", "prodNNExtra", "prodTests", "rightShiftTests", "runModWTests", "setDuringTest", "shiftTest", "stickyTests", "subMod2NTests", "sumNN", "testFunNN", "testFunSNN", "testMul", "testSqr"})
		sources[path] = []byte(source)
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
	const filter = "^(TestCmp|TestSet|TestAdd|TestSub|TestNatMul|TestNatSqr|TestMulRangeN|TestNLZ|TestShiftLeft|TestShiftRight|TestModW|TestMontgomery|TestExpNN|TestFibo|TestBit|TestSticky|TestNatSubMod2N|TestIssue37499|TestIssue42552)$"
	args := upstreamTestArgs("-test.run=" + filter)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			var result programResult
			if target == "vm" {
				result = upstreamOutcome(t, dir, nil, upstreamLimit(), "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "TestBigNatural", pkg, filter, target, dir, result.Text, result.Exit, true)
		})
	}
}
