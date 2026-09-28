package driver_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/load"
)

func TestHTTPCancellationUpstream(t *testing.T) {
	dir := filepath.Join(runtime.GOROOT(), "src/net/http")
	names := []string{"cancelTest", "runCancelTestTransport", "runCancelTestChannel", "runCancelTestContext", "runCancelTest", "TestTransportCancelRequest", "testTransportCancelRequest", "TestTransportCancelRequestInDo", "testTransportCancelRequestInDo", "TestTransportCancelRequestWithBodyInDo", "TestTransportCancelRequestWithBody", "testTransportCancelRequestWithBody", "TestTransportClosesRequestBody", "testTransportClosesRequestBody", "countCloseReader", "TestTransportCancelRequestBeforeDo", "testTransportCancelRequestBeforeDo", "TestTransportCancelRequestBeforeResponseHeaders", "testTransportCancelRequestBeforeResponseHeaders", "TestTransportCloseResponseBody", "testTransportCloseResponseBody", "TestTransportClosesBodyOnError", "testTransportClosesBodyOnError", "closerFunc", "closerFunc.Close", "TestTransportDialContext", "testTransportDialContext", "TestTransportFlushesBodyChunks", "logWritesConn", "TestTransportFlushesRequestHeader", "testTransportFlushesRequestHeader"}
	files, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	sources := load.Filter{}
	for _, path := range files {
		switch filepath.Base(path) {
		case "transport_test.go":
			sources[path] = []byte(selectedUpstreamSource(t, path, names))
		case "serve_test.go":
			sources[path] = []byte(strings.Replace(selectedUpstreamSource(t, path, []string{"neverEnding", "neverEnding.Read"}), ". \"net/http\"\n", "", 1))
		case "main_test.go":
			sources[path] = []byte(selectedUpstreamSource(t, path, []string{"TestMain", "interestingGoroutines", "goroutineLeaked", "runningBenchmarks", "waitCondition", "afterTest", "leakReported"}))
		case "export_test.go":
			source := selectedUpstreamSource(t, path, []string{"Transport.NumPendingRequestsForTesting", "MaxWriteWaitBeforeConnReuse"})
			values := timeExportSource(t, path, []string{"ExportErrRequestCanceled", "ExportErrRequestCanceledConn"})
			_, values, ok := strings.Cut(values, "package http\n")
			if !ok {
				t.Fatal("missing export package")
			}
			sources[path] = []byte(source + "\n" + values)
		case "clientserver_test.go":
			adapter, err := os.ReadFile("testdata/http-upstream-adapter.txt")
			if err != nil {
				t.Fatal(err)
			}
			sources[path] = adapter
		default:
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
	const filter = "^(TestTransportCancelRequest|TestTransportCancelRequestWithBody|TestTransportCancelRequestInDo|TestTransportCancelRequestWithBodyInDo|TestTransportClosesRequestBody|TestTransportCancelRequestBeforeDo|TestTransportCancelRequestBeforeResponseHeaders|TestTransportCloseResponseBody|TestTransportClosesBodyOnError|TestTransportDialContext|TestTransportFlushesBodyChunks|TestTransportFlushesRequestHeader)$/^(h1|h2)$"
	args := []string{"-test.v", "-test.short=" + strconv.FormatBool(testing.Short()), "-test.run=" + filter}
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			var got programResult
			if target == "vm" {
				got = programOutcome(t, dir, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				got = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "http-cancellation", "net/http", filter, target, dir, got.Text, got.Exit)
		})
	}
}
