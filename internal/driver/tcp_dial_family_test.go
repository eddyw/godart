package driver_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestTCPDialFamilyUpstream(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("the TCP fixture uses the Unix error helper")
	}
	dir, out := t.TempDir(), t.TempDir()
	netDir := filepath.Join(runtime.GOROOT(), "src", "net")
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/dialfamily\n\ngo 1.27\n\nrequire example.com/netpoll v0.0.0\nreplace example.com/netpoll => ./netpoll\n")
	for _, item := range []upstreamFile{
		{Name: "dial_test.go", Declarations: []string{"TestDialLocal"}},
		{Name: "error_test.go", Declarations: []string{"OpError.isValid", "parseDialError", "TestDialAddrError"}},
		{Name: "error_unix_test.go", Declarations: []string{"isPlatformError"}},
		{Name: "net.go", Declarations: []string{"noDeadline", "errTimeout", "timeoutError", "timeoutError.Error", "timeoutError.Timeout", "timeoutError.Temporary", "timeoutError.Is", "errMissingAddress", "errNoSuitableAddress", "errCanceled", "canceledError"}},
		{Name: "pipe.go", Declarations: []string{"pipeAddr", "pipeAddr.Network", "pipeAddr.String"}},
		{Name: "file.go", Declarations: []string{"fileAddr", "fileAddr.Network", "fileAddr.String"}},
	} {
		source := selectedUpstreamSource(t, filepath.Join(netDir, item.Name), item.Declarations)
		declaration := "package net_test\n"
		if item.Name == "dial_test.go" || item.Name == "error_test.go" {
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
	writeFile(t, filepath.Join(dir, "listener_test.go"), `package net_test
import (. "net"; "context"; "errors"; "net/netip"; "os"; "syscall"; "testing"; "time")
func newLocalListener(t testing.TB, network string) Listener {
 t.Helper()
 listener, err := Listen(network, "127.0.0.1:0")
 if err != nil { t.Fatal(err) }
 return listener
}
func supportsIPv4() bool { return supportsTCP("tcp4", "127.0.0.1:0") }
func supportsIPv6() bool { return supportsTCP("tcp6", "[::1]:0") }
func supportsTCP(network, address string) bool {
 listener, err := Listen(network, address)
 if err != nil { return false }
 listener.Close()
 return true
}
func TestTCPDialRefused(t *testing.T) {
 for _, family := range []struct{network, address string}{{"tcp", "127.0.0.1:0"}, {"tcp4", "127.0.0.1:0"}, {"tcp6", "[::1]:0"}} {
  t.Run(family.network, func(t *testing.T) {
   listener, err := Listen(family.network, family.address)
   if err != nil { t.Fatal(err) }
   defer listener.Close()
   address := listener.Addr()
   c, err := Dial(family.network, address.String())
   if err != nil { t.Fatal(err) }
   defer c.Close()
   peer, err := listener.Accept()
   if err != nil { t.Fatal(err) }
   defer peer.Close()
   if err := listener.Close(); err != nil { t.Fatal(err) }
   for _, api := range []string{"dial", "context", "timeout"} {
    t.Run(api, func(t *testing.T) {
     var c Conn
     var err error
     switch api {
     case "dial": c, err = Dial(family.network, address.String())
     case "context":
      ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
      defer cancel()
      var d Dialer
      c, err = d.DialContext(ctx, family.network, address.String())
     case "timeout": c, err = DialTimeout(family.network, address.String(), 5*time.Second)
     }
     if err == nil { c.Close(); t.Fatal("dial to a closed listener succeeded") }
     var op *OpError
     var call *os.SyscallError
     if !errors.Is(err, syscall.ECONNREFUSED) || !errors.As(err, &op) || !errors.As(err, &call) {
      t.Fatalf("error = %T %v, want OpError and SyscallError with ECONNREFUSED", err, err)
     }
     if op.Op != "dial" || op.Net != family.network || op.Source != nil || op.Addr == nil || op.Addr.String() != address.String() || call.Syscall != "connect" {
      t.Fatalf("error metadata = %+v, syscall = %s", op, call.Syscall)
     }
    })
   }
  })
 }
}
func TestTCPDialFamilies(t *testing.T) {
 for _, family := range []struct{ network, address string }{{"tcp4", "127.0.0.1:0"}, {"tcp6", "[::1]:0"}} {
  t.Run(family.network, func(t *testing.T) {
   listener, err := Listen(family.network, family.address)
   if err != nil { t.Fatal(err) }
   defer listener.Close()
   remote := listener.Addr().(*TCPAddr)
   for _, network := range []string{"tcp", family.network} {
    apis := []string{"dial", "context", "typed", "typed-context"}
    if network == family.network { apis = append(apis, "empty", "host", "wildcard") }
    for _, api := range apis {
     t.Run(network+"/"+api, func(t *testing.T) {
      ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
      defer cancel()
      var c Conn
      var err error
      var d Dialer
      switch api {
      case "dial": c, err = Dial(network, remote.String())
      case "context": c, err = d.DialContext(ctx, network, remote.String())
      case "empty", "host", "wildcard":
       host := ""
       if api == "host" { host = "localhost" }
       if api == "wildcard" { host = "::" }
       _, port, e := SplitHostPort(remote.String())
       if e != nil { t.Fatal(e) }
       c, err = d.DialContext(ctx, network, JoinHostPort(host, port))
      case "typed": c, err = DialTCP(network, nil, remote)
      case "typed-context": c, err = d.DialTCP(ctx, network, netip.AddrPort{}, remote.AddrPort())
      }
      if err != nil { t.Fatal(err) }
      defer c.Close()
      peer, err := listener.Accept()
      if err != nil { t.Fatal(err) }
      defer peer.Close()
      if c.RemoteAddr().String() != remote.String() { t.Fatalf("remote = %v, want %v", c.RemoteAddr(), remote) }
      if err := c.SetDeadline(time.Now().Add(5*time.Second)); err != nil { t.Fatal(err) }
      if err := peer.SetDeadline(time.Now().Add(5*time.Second)); err != nil { t.Fatal(err) }
      if _, err := c.Write([]byte{42}); err != nil { t.Fatal(err) }
      b := make([]byte, 1)
      if n, err := peer.Read(b); n != 1 || err != nil || b[0] != 42 { t.Fatalf("read = %d %v %v", n, b, err) }
     })
    }
   }
  })
 }
 for _, family := range []struct{network, host string}{{"tcp4", "::1"}, {"tcp6", "127.0.0.1"}, {"tcp6", "::ffff:127.0.0.1"}} {
  apis := []string{"dial"}
  if family.network == "tcp4" { apis = append(apis, "typed", "typed-context") }
  for _, api := range apis {
   t.Run("reject/"+family.network+"/"+family.host+"/"+api, func(t *testing.T) {
    addr := &TCPAddr{IP: ParseIP(family.host)}
    var c Conn
    var err error
    var d Dialer
    switch api {
    case "dial": c, err = Dial(family.network, addr.String())
    case "typed": c, err = DialTCP(family.network, nil, addr)
    case "typed-context": c, err = d.DialTCP(context.Background(), family.network, netip.AddrPort{}, addr.AddrPort())
    }
    if err == nil { c.Close(); t.Fatal("wrong address family succeeded") }
    var op *OpError
    var address *AddrError
    if !errors.As(err, &op) || !errors.As(err, &address) || op.Op != "dial" || op.Net != family.network || address.Addr != addr.IP.String() {
     t.Fatalf("error = %T %v, want dial %s address error for %s", err, err, family.network, addr.IP)
    }
   })
  }
 }
}
`)
	if err := os.Mkdir(filepath.Join(dir, "netpoll"), 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "netpoll/go.mod"), "module example.com/netpoll\n\ngo 1.27\n")
	poll := filepath.Join(runtime.GOROOT(), "src", "internal", "poll", "fd.go")
	writeFile(t, filepath.Join(dir, "netpoll/errors.go"), selectedUpstreamSource(t, poll, []string{"DeadlineExceededError"}))
	license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
	writeFile(t, filepath.Join(dir, "netpoll/LICENSE"), string(license))
	t.Log(run(t, dir, "go", "test", "-run", "^TestTCPDial(Families|Refused)$", "-v", "."))
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TrimPath: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.Binaries) != 1 {
		t.Fatalf("got %d test binaries", len(stats.Binaries))
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot"} {
		t.Run("families/"+target, func(t *testing.T) {
			args := []string{"-test.v", "-test.run=^TestTCPDial(Families|Refused)$"}
			var result programResult
			if target == "vm" {
				result = programOutcome(t, dir, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			t.Log(result.Text)
			if result.Exit != 0 {
				t.Fatal(result.Text)
			}
		})
	}
	const filter = "^(TestDialLocal|TestDialAddrError)$"
	args := []string{"-test.v", "-test.short=" + strconv.FormatBool(testing.Short()), "-test.run=" + filter}
	result := programOutcome(t, netDir, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
	compareUpstream(t, "net-tcp-dial-family", "net", filter, "vm", netDir, result.Text, result.Exit)
	t.Run("aot", func(t *testing.T) {
		result := runUpstreamTarget(t, "aot", out, stats.Binaries[0].Entry, netDir, args)
		compareUpstream(t, "net-tcp-dial-family", "net", filter, "aot", netDir, result.Text, result.Exit)
	})
}
