package driver_test

import (
	"encoding/json"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/load"
	"github.com/eddyw/godart/internal/parity"
)

func TestResolverLifecycle(t *testing.T) {
	dir := filepath.Join(runtime.GOROOT(), "src/net")
	tests := []string{"TestDNSTimeout", "TestIPVersion", "TestLookupIPAddrPreservesContextValues", "TestLookupIPAddrConcurrentCallsForNetworks", "TestResolverLookupIPWithEmptyHost", "TestWithUnexpiredValuesPreserved", "TestReverseAddress", "TestDNSErrorUnwrap"}
	names := map[string][]string{"lookup_test.go": append([]string{"ipVersionTests", "revAddrTests", "goResolver"}, tests...)}
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
			selected := tests
			var exclusions map[string]string
			filter := "^(" + strings.Join(selected, "|") + ")$"
			args := upstreamTestArgs("-test.run=" + filter)
			var result programResult
			if target == "vm" {
				result = programOutcome(t, dir, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstreamInventory(t, "net-resolver-lifecycle", "net", filter, target, dir, result.Text, result.Exit, inventory, nil, nil, exclusions, true)
		})
	}
}
