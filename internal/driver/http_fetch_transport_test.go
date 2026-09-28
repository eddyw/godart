package driver_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestHTTPFetchTransport(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	origin := filepath.Join(runtime.GOROOT(), "src/net/http")
	source := selectedUpstreamSource(t, filepath.Join(origin, "transport_test.go"), []string{"doneContext", "doneContext.Done", "doneContext.Err", "TestTransportCheckContextDoneEarly", "bodyCloser", "TestTransportClosesBodyOnInvalidRequests", "testTransportClosesBodyOnInvalidRequests", "fooProto", "cancelProto", "errCancelProto", "TestTransportAltProto", "TestAltProtoCancellation", "TestTransportEmptyMethod"})
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/httpvalidation\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "invalid_test.go"), source)
	license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
	// The adapter rejects I/O before a server can receive an invalid request.
	// These mode names preserve upstream cases, not HTTP/2 server coverage.
	writeFile(t, filepath.Join(dir, "adapter_test.go"), `package http_test
import("context";"errors";"net";. "net/http";"runtime";"testing")
type testMode string
type fixtureServer struct {URL string; handler Handler}
type clientServerTest struct {ts *fixtureServer}
func (s *fixtureServer) Client() *Client {
 tr:=&Transport{}
 // Web cases must keep the fetch path.
 if runtime.GOOS!="js" {
  tr.DialContext=func(context.Context,string,string)(net.Conn,error){
   s.handler.ServeHTTP(nil,nil)
   return nil,errors.New("unexpected dial")
  }
 }
 return &Client{Transport:tr}
}
func newClientServerTest(_ *testing.T,mode testMode,h Handler)*clientServerTest {
 scheme:="http";if mode=="h2" {scheme="https"}
 return &clientServerTest{ts:&fixtureServer{URL:scheme+"://127.0.0.1:1",handler:h}}
}
func run(t *testing.T,f func(*testing.T,testMode)) {
 for _,mode:=range []testMode{"h1","h2"} {t.Run(string(mode),func(t *testing.T){f(t,mode)})}
}
`)
	helpers := selectedUpstreamSource(t, filepath.Join(origin, "main_test.go"), []string{"afterTest", "leakReported", "interestingGoroutines"})
	writeFile(t, filepath.Join(dir, "helpers_test.go"), helpers)
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.Binaries) != 1 {
		t.Fatal(stats.Binaries)
	}
	prepareDart(t, out)
	prelude := filepath.Join(out, "fetch-denied.mjs")
	writeFile(t, prelude, "globalThis.fetch = () => { console.error('unexpected fetch'); process.exit(97); };\n")
	args := upstreamTestArgs("-test.run=^(TestTransportCheckContextDoneEarly|TestTransportClosesBodyOnInvalidRequests|TestTransportAltProto|TestAltProtoCancellation|TestTransportEmptyMethod)$/^(h1|h2)$")
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			var result programResult
			if target == "vm" {
				result = upstreamOutcome(t, dir, nil, runLimit, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args, "--import", prelude)
			}
			compareUpstream(t, "http-fetch-transport", "net/http", "^(TestTransportCheckContextDoneEarly|TestTransportClosesBodyOnInvalidRequests|TestTransportAltProto|TestAltProtoCancellation|TestTransportEmptyMethod)$/^(h1|h2)$", target, origin, result.Text, result.Exit, false)
		})
	}
}
