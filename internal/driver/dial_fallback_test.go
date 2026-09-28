package driver_test

import (
	"encoding/json"
	"go/ast"
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

func TestDialFallbackUpstream(t *testing.T) {
	dir := filepath.Join(runtime.GOROOT(), "src/net")
	tests := []string{"TestDialParallel", "TestDialerPartialDeadline", "TestAddrListPartition", "TestDialerFallbackDelay", "TestDialParallelSpuriousConnection"}
	names := map[string][]string{
		"dial_test.go":       {"slowDst4", "slowDst6", "TestDialParallel", "TestDialerPartialDeadline", "TestDialerFallbackDelay", "TestDialParallelSpuriousConnection", "lookupSlowFast", "slowDialTCP"},
		"ipsock_test.go":     {"TestAddrListPartition"},
		"mockserver_test.go": {"newDualStackListener", "newDualStackServer", "dualStackServer", "streamListener"},
	}
	files, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	sources := load.Filter{}
	for _, path := range files {
		if declarations, ok := names[filepath.Base(path)]; ok {
			source := selectedUpstreamSource(t, path, declarations)
			if filepath.Base(path) == "mockserver_test.go" {
				fset := token.NewFileSet()
				file, err := parser.ParseFile(fset, path, source, 0)
				if err != nil {
					t.Fatal(err)
				}
				for i := len(file.Decls) - 1; i >= 0; i-- {
					fn, ok := file.Decls[i].(*ast.FuncDecl)
					if !ok || fn.Recv == nil {
						continue
					}
					receiver := fn.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name
					if receiver == "streamListener" {
						source = source[:fset.Position(fn.Pos()).Offset] + source[fset.Position(fn.End()).Offset:]
					}
				}
			}
			sources[path] = []byte(source)
		} else {
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.PackageClauseOnly)
			if err != nil {
				t.Fatal(err)
			}
			sources[path] = []byte("package " + file.Name.Name + "\n")
		}
	}
	sources[filepath.Join(dir, "net_test.go")] = []byte(selectedUpstreamSource(t, filepath.Join(dir, "net.go"), []string{"noDeadline"}))
	sources[filepath.Join(dir, "platform_test.go")] = []byte(`package net
 func supportsIPv4() bool { return supportsTCPFamily("tcp4", "127.0.0.1:0") }
 func supportsIPv6() bool { return supportsTCPFamily("tcp6", "[::1]:0") }
 func supportsTCPFamily(network, address string) bool {
  listener, err := Listen(network, address)
  if err != nil { return false }
  listener.Close()
  return true
 }
 `)
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
			if target == "js" || target == "wasm" {
				selected = tests[:3]
				exclusions = map[string]string{
					"TestDialerFallbackDelay":            "the web runtime has no TCP sockets",
					"TestDialParallelSpuriousConnection": "the web runtime has no TCP sockets",
				}
			}
			filter := "^(" + strings.Join(selected, "|") + ")$"
			args := upstreamTestArgs("-test.run=" + filter)
			var result programResult
			if target == "vm" {
				result = upstreamOutcome(t, dir, nil, runLimit, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstreamInventory(t, "net-dial-fallback", "net", filter, target, dir, result.Text, result.Exit, inventory, nil, nil, exclusions, true)
		})
	}
}
