package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestTLSUpstream(t *testing.T) {
	dir := filepath.Join(runtime.GOROOT(), "src/crypto/tls")
	out := t.TempDir()
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TrimPath: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.Binaries) != 1 {
		t.Fatalf("got %d test binaries", len(stats.Binaries))
	}
	prepareDart(t, out)
	const filter = "^(TestMarshalUnmarshal|TestFuzz|TestRejectEmptySCTList|TestRejectEmptySCT|TestRejectDuplicateExtensions|TestECHRemoveOuterPSK|TestSignatureSelection|TestLegacyTypeAndHash|TestSupportedSignatureAlgorithms|TestSplitPreMasterSecret|TestKeysFromPreMasterSecret|TestDecodeECHConfigLists|TestSkipBadConfigs|TestPickECHConfigWithInvalidAEADID|TestECHPadding|TestRoundUp|TestRemovePadding|TestDialTimeout|TestDeadlineOnWrite|TestDialer|TestTLSUniqueMatches|TestConnCloseBreakingWrite|TestConnCloseWrite|TestWarningAlertFlood|TestHairpinInClose|TestRecordBadVersionTLS13|TestKeyUpdateSpamPostHandshakeTLS13|TestACVPVectors|TestTrafficKey)$"
	args := upstreamTestArgs("-test.run=" + filter)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			var result programResult
			if target == "vm" {
				result = programOutcome(t, dir, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "tls-selected", "crypto/tls", filter, target, dir, result.Text, result.Exit)
		})
	}
}
