package driver_test

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/load"
)

func TestMaphashPortable(t *testing.T) {
	tests := []string{"TestUnseededHash", "TestSeededHash", "TestHashGrouping", "TestHashBytesVsString", "TestHashHighBytes", "TestRepeat", "TestSeedFromSum64", "TestSeedFromSeed", "TestSeedFromFlush", "TestSeedFromReset", "TestComparableShouldPanic", "TestWriteComparableNoncommute", "TestHashInterface"}
	const pkg = "hash/maphash"
	dir := filepath.Join(runtime.GOROOT(), "src", pkg)
	path := filepath.Join(dir, "maphash_test.go")
	sources := load.Filter{path: []byte(selectedUpstreamSource(t, path, tests))}
	distributionPath := filepath.Join(dir, "smhasher_test.go")
	distributionTests := []string{"TestSmhasherSanity", "TestSmhasherAppendedZeros", "TestSmhasherSmallKeys", "TestSmhasherZeros"}
	declarations := append([]string{"fixedSeed", "bytesHash", "stringHash", "hashSize", "randBytes", "hashSet", "newHashSet"}, distributionTests...)
	sources[distributionPath] = []byte(selectedUpstreamSource(t, distributionPath, declarations))
	tests = append(tests, distributionTests...)
	sources[filepath.Join(dir, "example_test.go")] = []byte("package maphash_test\n")
	sources[filepath.Join(dir, "example_bloom_test.go")] = []byte("package maphash_test\n")
	out := t.TempDir()
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TestSources: sources, TrimPath: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.Binaries) != 1 {
		t.Fatal(stats.Binaries)
	}
	prepareDart(t, out)
	filter := "^(" + strings.Join(tests, "|") + ")$"
	args := upstreamTestArgs("-test.run=" + filter)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			var result programResult
			if target == "vm" {
				result = upstreamOutcome(t, dir, nil, upstreamLimit(), "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "maphash-portable", pkg, filter, target, dir, result.Text, result.Exit, true)
		})
	}
}
