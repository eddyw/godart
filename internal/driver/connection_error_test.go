package driver_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestConnectionErrorsUpstream(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("the socket fixture uses Unix error helpers")
	}
	dir, out := t.TempDir(), t.TempDir()
	netDir := filepath.Join(runtime.GOROOT(), "src", "net")
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/connectionerrors\n\ngo 1.27\n\nrequire example.com/netpoll v0.0.0\nreplace example.com/netpoll => ./netpoll\n")
	for _, item := range []upstreamFile{
		{Name: "timeout_test.go", Declarations: []string{"TestReadTimeout", "readTimeoutTests", "TestReadFromTimeout", "readFromTimeoutTests", "TestReadTimeoutMustNotReturn", "TestWriteTimeout", "TestWriteTimeoutMustNotReturn", "TestWriteToTimeout", "writeTimeoutTests", "isDeadlineExceeded", "TestReadTimeoutFluctuation", "TestReadFromTimeoutFluctuation", "TestWriteTimeoutFluctuation", "minDynamicTimeout", "maxDynamicTimeout", "timeoutUpperBound", "nextTimeout", "TestReadWriteProlongedTimeout", "TestReadWriteDeadlineRace", "TestConcurrentSetDeadline"}},
		{Name: "error_test.go", Declarations: []string{"OpError.isValid", "parseReadError", "parseWriteError", "parseCommonError"}},
		{Name: "error_unix_test.go", Declarations: []string{"isPlatformError", "isENOBUFS"}},
		{Name: "net.go", Declarations: []string{"noDeadline", "errTimeout", "timeoutError", "timeoutError.Error", "timeoutError.Timeout", "timeoutError.Temporary", "timeoutError.Is", "errMissingAddress", "errCanceled", "canceledError"}},
		{Name: "pipe.go", Declarations: []string{"pipeAddr", "pipeAddr.Network", "pipeAddr.String"}},
		{Name: "file.go", Declarations: []string{"fileAddr", "fileAddr.Network", "fileAddr.String"}},
		{Name: "mockserver_test.go", Declarations: []string{"localServer.buildup", "localServer.teardown", "newLocalServer", "localPacketServer.buildup", "localPacketServer.teardown", "newLocalPacketServer"}},
	} {
		source := selectedUpstreamSource(t, filepath.Join(netDir, item.Name), item.Declarations)
		declaration := "package net_test\n"
		if item.Name == "timeout_test.go" || item.Name == "error_test.go" || item.Name == "mockserver_test.go" {
			declaration += "import . \"net\"\n"
		}
		source = strings.Replace(source, "package net\n", declaration, 1)
		// An external test package cannot define a method on net.OpError.
		source = strings.Replace(source, "func (e *OpError) isValid() error", "func validOpError(e *OpError) error", 1)
		source = strings.ReplaceAll(source, "err.isValid()", "validOpError(err)")
		source = strings.Replace(source, `ErrWriteToConnected = errors.New("use of WriteTo with pre-connected connection")`, "", 1)
		source = strings.ReplaceAll(source, `"internal/poll"`, `"example.com/netpoll"`)
		name := strings.TrimSuffix(item.Name, ".go") + "_test.go"
		writeFile(t, filepath.Join(dir, name), source)
	}

	serverSource, err := os.ReadFile(filepath.Join(netDir, "mockserver_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	fs := token.NewFileSet()
	file, err := parser.ParseFile(fs, "mockserver_test.go", serverSource, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	types := string(serverSource[:fs.Position(file.Name.Pos()).Offset]) + "net_test\nimport (. \"net\"; \"sync\")\n"
	found := 0
	for _, decl := range file.Decls {
		group, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range group.Specs {
			typ, ok := spec.(*ast.TypeSpec)
			if ok && (typ.Name.Name == "localServer" || typ.Name.Name == "localPacketServer" || typ.Name.Name == "streamListener" || typ.Name.Name == "packetListener") {
				types += string(serverSource[fs.Position(group.Pos()).Offset:fs.Position(group.End()).Offset]) + "\n"
				found++
			}
		}
	}
	if found != 4 {
		t.Fatalf("found %d upstream server types, want 4", found)
	}
	writeFile(t, filepath.Join(dir, "server_types_test.go"), types)
	writeFile(t, filepath.Join(dir, "listener_test.go"), `package net_test
import (. "net"; "testing")
func newLocalListener(t testing.TB, network string) Listener {
 t.Helper()
 listener, err := Listen(network, "127.0.0.1:0")
 if err != nil { t.Fatal(err) }
 return listener
}
func newLocalPacketListener(t testing.TB, network string) PacketConn {
 t.Helper()
 c,err:=ListenPacket(network,"127.0.0.1:0")
 if err!=nil{t.Fatal(err)}
 return c
}
`)
	if err := os.Mkdir(filepath.Join(dir, "netpoll"), 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "netpoll/go.mod"), "module example.com/netpoll\n\ngo 1.27\n")
	poll := filepath.Join(runtime.GOROOT(), "src", "internal", "poll", "fd.go")
	writeFile(t, filepath.Join(dir, "netpoll/errors.go"), selectedUpstreamSource(t, poll, []string{"ErrNotPollable", "DeadlineExceededError"}))
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
	const filter = "^(TestReadTimeout|TestReadFromTimeout|TestReadTimeoutMustNotReturn|TestWriteTimeout|TestWriteTimeoutMustNotReturn|TestWriteToTimeout|TestReadTimeoutFluctuation|TestReadFromTimeoutFluctuation|TestWriteTimeoutFluctuation|TestReadWriteProlongedTimeout|TestReadWriteDeadlineRace|TestConcurrentSetDeadline)$"
	args := []string{"-test.v", "-test.short=" + strconv.FormatBool(testing.Short()), "-test.run=" + filter}
	result := programOutcome(t, netDir, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
	compareUpstream(t, "net-connection-errors", "net", filter, "vm", netDir, result.Text, result.Exit)
	t.Run("aot", func(t *testing.T) {
		result := runUpstreamTarget(t, "aot", out, stats.Binaries[0].Entry, netDir, args)
		compareUpstream(t, "net-connection-errors", "net", filter, "aot", netDir, result.Text, result.Exit)
	})
}
