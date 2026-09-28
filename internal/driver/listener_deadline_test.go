package driver_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestListenerDeadlineUpstream(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("the listener fixture uses the Unix error helper")
	}
	dir, out := t.TempDir(), t.TempDir()
	netDir := filepath.Join(runtime.GOROOT(), "src", "net")
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/listener\n\ngo 1.27\n\nrequire example.com/netpoll v0.0.0\nreplace example.com/netpoll => ./netpoll\n")
	for _, item := range []upstreamFile{
		{Name: "timeout_test.go", Declarations: []string{"TestAcceptTimeout", "TestAcceptTimeoutMustReturn", "TestAcceptTimeoutMustNotReturn", "isDeadlineExceeded"}},
		{Name: "error_test.go", Declarations: []string{"OpError.isValid", "parseAcceptError"}},
		{Name: "error_unix_test.go", Declarations: []string{"isPlatformError"}},
		{Name: "net.go", Declarations: []string{"noDeadline", "errTimeout", "timeoutError", "timeoutError.Error", "timeoutError.Timeout", "timeoutError.Temporary", "timeoutError.Is"}},
		{Name: "pipe.go", Declarations: []string{"pipeAddr", "pipeAddr.Network", "pipeAddr.String"}},
		{Name: "file.go", Declarations: []string{"fileAddr", "fileAddr.Network", "fileAddr.String"}},
	} {
		source := selectedUpstreamSource(t, filepath.Join(netDir, item.Name), item.Declarations)
		declaration := "package net_test\n"
		if item.Name == "timeout_test.go" || item.Name == "error_test.go" {
			declaration += "import . \"net\"\n"
		}
		source = strings.Replace(source, "package net\n", declaration, 1)
		// An external test package cannot define a method on net.OpError.
		source = strings.Replace(source, "func (e *OpError) isValid() error", "func validOpError(e *OpError) error", 1)
		source = strings.ReplaceAll(source, "err.isValid()", "validOpError(err)")
		source = strings.ReplaceAll(source, `"internal/poll"`, `"example.com/netpoll"`)
		name := strings.TrimSuffix(item.Name, ".go") + "_test.go"
		writeFile(t, filepath.Join(dir, name), source)
	}
	writeFile(t, filepath.Join(dir, "listener_test.go"), `package net_test
import (. "net"; "testing")
func newLocalListener(t testing.TB, network string) Listener {
 t.Helper()
 listener, err := Listen(network, "127.0.0.1:0")
 if err != nil { t.Fatal(err) }
 return listener
}
`)
	if err := os.Mkdir(filepath.Join(dir, "netpoll"), 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "netpoll/go.mod"), "module example.com/netpoll\n\ngo 1.27\n")
	poll := filepath.Join(runtime.GOROOT(), "src", "internal", "poll", "fd.go")
	writeFile(t, filepath.Join(dir, "netpoll/errors.go"), selectedUpstreamSource(t, poll, []string{"ErrNotPollable"}))
	license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
	writeFile(t, filepath.Join(dir, "netpoll/LICENSE"), string(license))
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TrimPath: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.Binaries) != 1 {
		t.Fatalf("got %d test binaries", len(stats.Binaries))
	}
	prepareDart(t, out)
	const filter = "^(TestAcceptTimeout|TestAcceptTimeoutMustReturn|TestAcceptTimeoutMustNotReturn)$"
	args := upstreamTestArgs("-test.run=" + filter)
	result := upstreamOutcome(t, netDir, nil, runLimit, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
	compareUpstream(t, "net-listener-deadlines", "net", filter, "vm", netDir, result.Text, result.Exit, true)
	t.Run("aot", func(t *testing.T) {
		result := runUpstreamTarget(t, "aot", out, stats.Binaries[0].Entry, netDir, args)
		compareUpstream(t, "net-listener-deadlines", "net", filter, "aot", netDir, result.Text, result.Exit, true)
	})
}
