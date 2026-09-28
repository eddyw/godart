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

func TestResolverConfiguration(t *testing.T) {
	dir := filepath.Join(runtime.GOROOT(), "src/net")
	names := map[string][]string{
		"conf_test.go":           nil,
		"hosts_test.go":          nil,
		"lookup_test.go":         {"TestLookupPort", "TestLookupPort_Minimal", "TestLookupPortNotFound", "TestLookupPortDifferentNetwork", "TestLookupPortEmptyNetworkString", "TestLookupPortIPNetworkString", "tcpOnlyService"},
		"error_test.go":          {"parseLookupPortError"},
		"main_conf_test.go":      {"forceGoDNS", "forceCgoDNS", "TestForceGoDNS", "TestForceCgoDNS", "allResolvers"},
		"nss_test.go":            nil,
		"parse_test.go":          nil,
		"dnsconfig_unix_test.go": {"dnsReadConfigTests", "TestDNSReadConfig", "TestDNSReadMissingFile", "dnsDefaultSearchTests", "TestDNSDefaultSearch", "TestDNSNameLength"},
		"dnsclient_unix_test.go": {"resolvConfTest", "newResolvConfTest", "TestAvoidDNSName", "TestNameListAvoidDNS"},
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
	tests := []string{"TestLookupStaticHost", "TestLookupStaticAddr", "TestHostCacheModification", "TestLookupStaticHostAliases", "TestConfHostLookupOrder", "TestAddrLookupOrder", "TestSystemConf", "TestForceGoDNS", "TestForceCgoDNS", "TestParseNSSConf", "TestReadLine", "TestDtoi", "TestDNSReadConfig", "TestDNSReadMissingFile", "TestDNSDefaultSearch", "TestDNSNameLength", "TestAvoidDNSName", "TestNameListAvoidDNS", "TestLookupPort", "TestLookupPort_Minimal", "TestLookupPortNotFound", "TestLookupPortDifferentNetwork", "TestLookupPortEmptyNetworkString", "TestLookupPortIPNetworkString"}
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			selected := slices.Clone(tests)
			var exclusions map[string]string
			if target == "js" || target == "wasm" {
				exclusions = map[string]string{}
				for _, name := range []string{"TestLookupStaticHost", "TestLookupStaticAddr", "TestHostCacheModification", "TestLookupStaticHostAliases", "TestConfHostLookupOrder", "TestAddrLookupOrder", "TestParseNSSConf", "TestReadLine", "TestDNSReadConfig", "TestDNSReadMissingFile", "TestDNSNameLength"} {
					exclusions[name] = "The Dart web target has no host file access."
				}
				selected = slices.DeleteFunc(selected, func(name string) bool { return exclusions[name] != "" })
			}
			filter := "^(" + strings.Join(selected, "|") + ")$"
			args := upstreamTestArgs("-test.run=" + filter)
			var result programResult
			if target == "vm" {
				result = upstreamOutcome(t, dir, nil, runLimit, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstreamInventory(t, "net-resolver-config", "net", filter, target, dir, result.Text, result.Exit, inventory, nil, nil, exclusions, true)
		})
	}
}

func TestResolverPortPolicy(t *testing.T) {
	dir, err := filepath.Abs("../../testdata/programs/resolverports")
	if err != nil {
		t.Fatal(err)
	}
	for _, tag := range []string{"", "netgo", "netcgo"} {
		t.Run(tag, func(t *testing.T) {
			var tags []string
			native := filepath.Join(t.TempDir(), "program")
			args := []string{"build", "-o", native}
			if tag != "" {
				tags = []string{tag}
				args = append(args, "-tags="+tag)
			}
			run(t, dir, "go", append(args, ".")...)
			out := t.TempDir()
			if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), BuildTags: tags}); err != nil {
				t.Fatal(err)
			}
			prepareDart(t, out)
			for _, mode := range []string{"go", "cgo"} {
				t.Run(mode, func(t *testing.T) {
					t.Setenv("GODEBUG", "netdns="+mode)
					want := programReference(t, dir, native)
					got := programOutcome(t, out, "dart", "bin/main.dart")
					if got != want {
						t.Fatalf("resolver policy differs\nGo:\n%+v\nGart:\n%+v", want, got)
					}
				})
			}
		})
	}
}
