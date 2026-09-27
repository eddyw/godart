package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestHTTPContextBeforeFetch(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	origin := filepath.Join(runtime.GOROOT(), "src/net/http")
	source := selectedUpstreamSource(t, filepath.Join(origin, "transport_test.go"), []string{"doneContext", "doneContext.Done", "doneContext.Err", "TestTransportCheckContextDoneEarly"})
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/httpcontext\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "context_test.go"), source)
	license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.Binaries) != 1 {
		t.Fatal(stats.Binaries)
	}
	prepareDart(t, out)
	prelude := filepath.Join(out, "fetch-denied.mjs")
	writeFile(t, prelude, "globalThis.fetch = () => Promise.reject(new Error('unexpected fetch'));\n")
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			var result programResult
			if target == "vm" {
				result = programOutcome(t, dir, "dart", filepath.Join(out, stats.Binaries[0].Entry), "-test.v")
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, []string{"-test.v"}, "--import", prelude)
			}
			compareUpstream(t, "http-context-before-fetch", "net/http", "^TestTransportCheckContextDoneEarly$", target, origin, result.Text, result.Exit)
		})
	}
}
