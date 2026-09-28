package driver_test

import (
	"encoding/json"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/load"
	"github.com/eddyw/godart/internal/parity"
)

func TestResolverDNS(t *testing.T) {
	dir := filepath.Join(runtime.GOROOT(), "src/net")
	tests := []string{
		"TestDNSSRVUniformity",
		"TestWeighting",
		"TestDNSTransportFallback",
		"TestDNSTransportNoFallbackOnTCP",
		"TestSpecialDomainName",
		"TestAvoidDNSName",
		"TestNameListAvoidDNS",
		"TestLookupTorOnion",
		"TestUpdateResolvConf",
		"TestGoLookupIPWithResolverConfig",
		"TestGoLookupIPOrderFallbackToFile",
		"TestErrorForOriginalNameWhenSearching",
		"TestIgnoreLameReferrals",
		"TestIgnoreDNSForgeries",
		"TestRetryTimeout",
		"TestRotate",
		"TestStrictErrorsLookupIP",
		"TestStrictErrorsLookupTXT",
		"TestDNSGoroutineRace",
		"TestIssue8434",
		"TestIssueNoSuchHostExists",
		"TestNoSuchHost",
		"TestDNSDialTCP",
		"TestTXTRecordTwoStrings",
		"TestSingleRequestLookup",
		"TestDNSUseTCP",
		"TestDNSUseTCPTruncated",
		"TestPTRandNonPTR",
		"TestCVE202133195",
		"TestNullMX",
		"TestRootNS",
		"TestGoLookupIPCNAMEOrderHostsAliasesFilesOnlyMode",
		"TestGoLookupIPCNAMEOrderHostsAliasesFilesDNSMode",
		"TestGoLookupIPCNAMEOrderHostsAliasesDNSFilesMode",
		"TestDNSPacketSize",
		"TestLongDNSNames",
		"TestDNSTrustAD",
		"TestDNSConfigNoReload",
		"TestLookupOrderFilesNoSuchHost",
		"TestExtendedRCode",
		"TestEmptyResolvConfReplacedWithConfHaingNameservers",
	}
	names := map[string][]string{
		"dnsclient_unix_test.go": nil,
		"dnsclient_test.go":      nil,
		"hosts_test.go":          {"lookupStaticHostAliasesTest"},
		"conf_test.go":           {"nssStr", "setSystemNSS"},
	}
	files, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	sources := load.Filter{}
	for _, path := range files {
		if declarations, ok := names[filepath.Base(path)]; ok {
			sources[path] = []byte(selectedUpstreamSource(t, path, declarations))
		} else {
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.PackageClauseOnly)
			if err != nil {
				t.Fatal(err)
			}
			sources[path] = []byte("package " + file.Name.Name + "\n")
		}
	}
	out := t.TempDir()
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TestSources: sources, TrimPath: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.Binaries) != 1 {
		t.Fatalf("got %d test binaries", len(stats.Binaries))
	}
	prepareDart(t, out)
	inventoryName, err := parity.InventoryName(build.Default)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join("..", "parity", "testdata", inventoryName))
	if err != nil {
		t.Fatal(err)
	}
	var inventory parity.Inventory
	if err := json.Unmarshal(data, &inventory); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			selected := slices.Clone(tests)
			var exclusions map[string]string
			if target == "js" || target == "wasm" {
				exclusions = map[string]string{
					"TestUpdateResolvConf":                                "The Dart web target has no host file access.",
					"TestGoLookupIPWithResolverConfig":                    "The Dart web target has no host file access.",
					"TestGoLookupIPOrderFallbackToFile":                   "The Dart web target has no host file access.",
					"TestErrorForOriginalNameWhenSearching":               "The Dart web target has no host file access.",
					"TestIgnoreLameReferrals":                             "The Dart web target has no host file access.",
					"TestRetryTimeout":                                    "The Dart web target has no host file access.",
					"TestStrictErrorsLookupIP":                            "The Dart web target has no host file access.",
					"TestStrictErrorsLookupTXT":                           "The Dart web target has no host file access.",
					"TestSingleRequestLookup":                             "The Dart web target has no host file access.",
					"TestGoLookupIPCNAMEOrderHostsAliasesFilesOnlyMode":   "The Dart web target has no host file access.",
					"TestGoLookupIPCNAMEOrderHostsAliasesFilesDNSMode":    "The Dart web target has no host file access.",
					"TestGoLookupIPCNAMEOrderHostsAliasesDNSFilesMode":    "The Dart web target has no host file access.",
					"TestDNSTrustAD":                                      "The Dart web target has no host file access.",
					"TestDNSConfigNoReload":                               "The Dart web target has no host file access.",
					"TestLookupOrderFilesNoSuchHost":                      "The Dart web target has no host file access.",
					"TestEmptyResolvConfReplacedWithConfHaingNameservers": "The Dart web target has no host file access.",
					"TestRotate": "The Dart web target has no host file access.",
				}
				selected = slices.DeleteFunc(selected, func(name string) bool { return exclusions[name] != "" })
			}
			filter := "^(" + strings.Join(selected, "|") + ")$"
			args := upstreamTestArgs("-test.run=" + filter)
			var result programResult
			if target == "vm" {
				result = programOutcome(t, dir, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstreamInventory(t, "net-resolver-dns", "net", filter, target, dir, result.Text, result.Exit, inventory, nil, nil, exclusions, true)
		})
	}
}
