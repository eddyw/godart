package driver_test

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestBigmodPortableArithmetic(t *testing.T) {
	const pkg = "crypto/internal/fips140/bigmod"
	dir := filepath.Join(runtime.GOROOT(), "src", pkg)
	tests := []string{"TestModAddCommutative", "TestModSubThenAddIdentity", "TestMontgomeryRoundtrip", "TestShiftIn", "TestModulusAndNatSizes", "TestSetBytes", "TestExpand", "TestMod", "TestModSub", "TestModAdd", "TestExp", "TestExpShort", "TestMulReductions", "TestMul", "TestIs", "TestTrailingZeroBits", "TestRightShift", "TestNewModulus", "TestAddMulVVWSized"}
	out := t.TempDir()
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TrimPath: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.Binaries) != 1 {
		t.Fatal(stats.Binaries)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			selected := tests
			if target == "vm" || target == "aot" {
				selected = append(append([]string{}, tests...), "TestInverse")
			}
			filter := "^(" + strings.Join(selected, "|") + ")$"
			args := upstreamTestArgs("-test.run=" + filter)
			var result programResult
			if target == "vm" {
				result = upstreamOutcome(t, dir, nil, upstreamLimit(), "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "bigmod-arithmetic", pkg, filter, target, dir, result.Text, result.Exit, true)
		})
	}
}
