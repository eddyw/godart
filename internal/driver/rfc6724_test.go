package driver_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestRFC6724(t *testing.T) {
	netDir := filepath.Join(runtime.GOROOT(), "src", "net")
	origin, err := os.ReadFile(filepath.Join(netDir, "addrselect.go"))
	if err != nil {
		t.Fatal(err)
	}
	overlay, err := os.ReadFile("../overlay/std/net/addrselect.go")
	if err != nil {
		t.Fatal(err)
	}
	source := strings.Replace(string(overlay), "//go:build dart\n\n", "", 1)
	if source != string(origin) {
		t.Fatal("the RFC 6724 overlay differs from the active Go SDK")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/rfc6724\n\ngo 1.27\n")
	source = strings.Replace(source, "package net\n", "package net_test\nimport . \"net\"\n", 1)
	writeFile(t, filepath.Join(dir, "addrselect_source_test.go"), source)
	tests := selectedUpstreamSource(t, filepath.Join(netDir, "addrselect_test.go"), nil)
	tests = strings.Replace(tests, "package net\n", "package net_test\nimport . \"net\"\n", 1)
	writeFile(t, filepath.Join(dir, "addrselect_test.go"), tests)
	license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TrimPath: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.Binaries) != 1 {
		t.Fatalf("got %d test binaries", len(stats.Binaries))
	}
	prepareDart(t, out)
	const filter = "^(TestSortByRFC6724|TestRFC6724PolicyTableOrder|TestRFC6724PolicyTableContent|TestRFC6724PolicyTableClassify|TestRFC6724ClassifyScope|TestRFC6724CommonPrefixLength)$"
	args := upstreamTestArgs("-test.run=" + filter)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			var result programResult
			if target == "vm" {
				result = programOutcome(t, netDir, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, netDir, args)
			}
			compareUpstream(t, "net-rfc6724", "net", filter, target, netDir, result.Text, result.Exit)
		})
	}
}
