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

func TestSignalDeliveryUpstream(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native signal validation requires macOS or Linux")
	}
	dir := filepath.Join(runtime.GOROOT(), "src/os/signal")
	names := []string{"TestSignal", "TestStress", "TestTime", "TestReset", "TestIgnore", "TestNotifyContextCause", "testCancel", "waitSig", "waitSigAll", "waitSig1", "quiesce", "settleTime", "fatalWaitingTime", "init", "TestStop", "sendUncaughtSighup", "dieFromSighup"}
	files, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	sources := load.Filter{}
	for _, path := range files {
		if filepath.Base(path) == "signal_test.go" {
			sources[path] = []byte(selectedUpstreamSource(t, path, names))
		} else {
			f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.PackageClauseOnly)
			if err != nil {
				t.Fatal(err)
			}
			sources[path] = []byte("package " + f.Name.Name + "\n")
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
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			for _, suite := range []struct{ name, filter string }{
				{"signal-delivery", "^(TestSignal|TestStress|TestTime|TestReset|TestIgnore|TestNotifyContextCause)$"},
				{"signal-stop", "^TestStop$"},
			} {
				t.Run(suite.name, func(t *testing.T) {
					args := upstreamTestArgs("-test.run=" + suite.filter)
					var got programResult
					if target == "vm" {
						got = upstreamOutcome(t, dir, nil, runLimit, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
					} else {
						got = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
					}
					compareUpstream(t, suite.name, "os/signal", suite.filter, target, dir, got.Text, got.Exit, true)
				})
			}
		})
	}
}
