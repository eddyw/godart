package driver_test

import (
	"os"
	"path/filepath"
	"runtime"
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
		{Name: "dial_test.go", Declarations: []string{"TestDialLocal", "TestDialerLocalAddr", "TestCancelAfterDial", "TestDialWithNonZeroDeadline", "contextWithNonZeroDeadline", "contextWithNonZeroDeadline.Deadline"}},
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
		if item.Name == "dial_test.go" {
			for _, line := range []string{
				"\torigTestHookLookupIP := testHookLookupIP\n",
				"\tdefer func() { testHookLookupIP = origTestHookLookupIP }()\n",
				"\ttestHookLookupIP = lookupLocalhost\n",
			} {
				if strings.Count(source, line) != 1 {
					t.Fatalf("the original local-address resolver setup changed: %q", line)
				}
				source = strings.Replace(source, line, "", 1)
			}
		}

		// An external test package cannot define a method on net.OpError.
		source = strings.Replace(source, "func (e *OpError) isValid() error", "func validOpError(e *OpError) error", 1)
		source = strings.ReplaceAll(source, "err.isValid()", "validOpError(err)")
		source = strings.Replace(source, `ErrWriteToConnected = errors.New("use of WriteTo with pre-connected connection")`, "", 1)
		source = strings.ReplaceAll(source, `"internal/poll"`, `"example.com/netpoll"`)
		name := strings.TrimSuffix(item.Name, ".go") + "_test.go"
		writeFile(t, filepath.Join(dir, name), source)
	}
	writeFile(t, filepath.Join(dir, "listener_test.go"), `package net_test
import (. "net"; "context"; "errors"; "fmt"; "net/netip"; "os"; "strconv"; "sync"; "syscall"; "testing"; "time")
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
type localServer struct { Listener; done sync.WaitGroup }
func newLocalServer(t testing.TB, network string) *localServer {
 t.Helper()
 address := "127.0.0.1:0"
 if network == "tcp6" { address = "[::1]:0" }
 listener, err := Listen(network, address)
 if err != nil { t.Fatal(err) }
 return &localServer{Listener: listener}
}
func (s *localServer) buildup(handler func(*localServer, Listener)) error {
 s.done.Add(1)
 go func() { defer s.done.Done(); handler(s, s.Listener) }()
 return nil
}
func (s *localServer) teardown() { s.Close(); s.done.Wait() }
func supportsIPv4map() bool {
 listener, err := Listen("tcp", "[::]:0")
 if err != nil { return false }
 defer listener.Close()
 _, port, err := SplitHostPort(listener.Addr().String())
 if err != nil { return false }
 c, err := DialTimeout("tcp4", JoinHostPort("127.0.0.1", port), 5*time.Second)
 if err != nil { return false }
 defer c.Close()
 peer, err := listener.Accept()
 if err != nil { return false }
 peer.Close()
 return true
}
func isEADDRINUSE(err error) bool { return errors.Is(err, syscall.EADDRINUSE) }
func TestTCPDialCanceled(t *testing.T) {
 for _, family := range []struct{network, address string}{{"tcp", "127.0.0.1:0"}, {"tcp4", "127.0.0.1:0"}, {"tcp6", "[::1]:0"}} {
  for _, mode := range []string{"context", "cancel", "deadline", "negative-timeout"} {
   for _, api := range []string{"context", "typed"} {
    t.Run(family.network+"/"+mode+"/"+api, func(t *testing.T) {
     listener, err := Listen(family.network, family.address)
     if err != nil { t.Fatal(err) }
     defer listener.Close()
     remote := listener.Addr().(*TCPAddr)
     local := &TCPAddr{IP: remote.IP}
     d := Dialer{LocalAddr: local}
     ctx := context.Background()
     var want error = context.Canceled
     switch mode {
     case "context":
      var cancel context.CancelFunc
      ctx, cancel = context.WithCancel(ctx)
      cancel()
     case "cancel":
      cancel := make(chan struct{})
      close(cancel)
      d.Cancel = cancel
     case "deadline": d.Deadline = time.Now().Add(-time.Second); want = context.DeadlineExceeded
     case "negative-timeout": d.Timeout = -time.Second; want = context.DeadlineExceeded
     }
     var c Conn
     if api == "context" { c, err = d.DialContext(ctx, family.network, remote.String()) } else {
      c, err = d.DialTCP(ctx, family.network, local.AddrPort(), remote.AddrPort())
     }
     if err == nil { c.Close(); if mode == "cancel" { return }; t.Fatal("canceled dial succeeded") }
     var op *OpError
     if !errors.Is(err, want) || !errors.As(err, &op) { t.Fatalf("error = %T %v, want OpError and %v", err, err, want) }
     if op.Op != "dial" || op.Net != family.network || op.Source == nil || op.Source.String() != local.String() || op.Addr == nil || op.Addr.String() != remote.String() { t.Fatalf("error metadata = %+v", op) }
     errorType := "net.canceledError"
     if want == context.DeadlineExceeded { errorType = "*net.timeoutError" }
     if got := fmt.Sprintf("%T", op.Err); got != errorType { t.Fatalf("error type = %s, want %s", got, errorType) }
     if op.Timeout() != (want == context.DeadlineExceeded) { t.Fatalf("timeout = %v for %v", op.Timeout(), want) }
     if api == "context" && mode != "cancel" {
      listener.(*TCPListener).SetDeadline(time.Now().Add(20*time.Millisecond))
      peer, err := listener.Accept()
      if err == nil { peer.Close(); t.Fatal("canceled dial opened a connection") }
      var nerr Error
      if !errors.As(err, &nerr) || !nerr.Timeout() { t.Fatalf("accept = %v, want timeout", err) }
     }
    })
   }
  }
 }
}
func TestTCPDialPorts(t *testing.T) {
 for _, family := range []struct{network, host string}{{"tcp", "127.0.0.1"}, {"tcp4", "127.0.0.1"}, {"tcp6", "::1"}} {
  t.Run(family.network, func(t *testing.T) {
   for _, service := range []string{"-1", "65536", "123456789", "--badport--"} {
    for _, api := range []string{"dial", "context"} {
     t.Run(service+"/"+api, func(t *testing.T) {
      address := JoinHostPort(family.host, service)
      var c Conn
      var err error
      if api == "dial" { c, err = Dial(family.network, address) } else {
       d := Dialer{Timeout: 5*time.Second}
       c, err = d.DialContext(context.Background(), family.network, address)
      }
      if err == nil { c.Close(); t.Fatal("invalid port succeeded") }
      var op *OpError
      if !errors.As(err, &op) || op.Op != "dial" || op.Net != family.network || op.Source != nil || op.Addr != nil { t.Fatalf("error metadata = %T %v", err, err) }
      if service == "--badport--" {
       var dns *DNSError
       if !errors.As(err, &dns) || dns.Name != family.network+"/"+service || !dns.IsNotFound { t.Fatalf("error = %T %v, want unknown service", err, err) }
      } else {
       var addr *AddrError
       if !errors.As(err, &addr) || addr.Err != "invalid port" || addr.Addr != service { t.Fatalf("error = %T %v, want invalid port %s", err, err, service) }
      }
     })
    }
   }
   for _, port := range []int{-1, 65536} {
    for _, invalidLocal := range []bool{false, true} {
     t.Run("typed/"+strconv.Itoa(port)+"/"+strconv.FormatBool(invalidLocal), func(t *testing.T) {
      var local *TCPAddr
      operation := "connect"
      if invalidLocal { local = &TCPAddr{IP: ParseIP(family.host), Port: -1}; operation = "bind" }
      remote := &TCPAddr{IP: ParseIP(family.host), Port: port}
      c, err := DialTCP(family.network, local, remote)
      if err == nil { c.Close(); t.Fatal("invalid port succeeded") }
      var op *OpError
      var call *os.SyscallError
      if !errors.Is(err, syscall.EINVAL) || !errors.As(err, &op) || !errors.As(err, &call) || call.Syscall != operation { t.Fatalf("error = %T %v, want %s EINVAL", err, err, operation) }
      if op.Op != "dial" || op.Net != family.network || op.Addr == nil || op.Addr.String() != remote.String() || (op.Source == nil) != (local == nil) { t.Fatalf("error metadata = %+v", op) }
      if local != nil && op.Source.String() != local.String() { t.Fatalf("source = %v, want %v", op.Source, local) }
     })
    }
   }
   t.Run("service", func(t *testing.T) {
    port, err := LookupPort(family.network, "svn")
    if err != nil { t.Fatal(err) }
    listener, err := Listen(family.network, JoinHostPort(family.host, strconv.Itoa(port)))
    if errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EADDRINUSE) { t.Skipf("local service port unavailable: %v", err) }
    if err != nil { t.Fatal(err) }
    defer listener.Close()
    for _, api := range []string{"dial", "context"} {
     t.Run(api, func(t *testing.T) {
      address := JoinHostPort(family.host, "svn")
      var c Conn
      var err error
      if api == "dial" { c, err = Dial(family.network, address) } else {
       d := Dialer{Timeout: 5*time.Second}
       c, err = d.DialContext(context.Background(), family.network, address)
      }
      if err != nil { t.Fatal(err) }
      defer c.Close()
      peer, err := listener.Accept()
      if err != nil { t.Fatal(err) }
      defer peer.Close()
      if c.RemoteAddr().String() != listener.Addr().String() { t.Fatalf("remote address = %v, want %v", c.RemoteAddr(), listener.Addr()) }
     })
    }
   })
  })
 }
}
func TestTCPDialLocalBinding(t *testing.T) {
 for _, family := range []struct{network, address string}{{"tcp4", "127.0.0.1:0"}, {"tcp6", "[::1]:0"}} {
  t.Run(family.network, func(t *testing.T) {
   listener, err := Listen(family.network, family.address)
   if err != nil { t.Fatal(err) }
   defer listener.Close()
   remote := listener.Addr().(*TCPAddr)
   for _, api := range []string{"dial", "context", "typed", "typed-context"} {
    for _, mode := range []string{"ephemeral", "fixed", "empty"} {
     t.Run(api+"/"+mode, func(t *testing.T) {
      local := &TCPAddr{IP: remote.IP}
      if mode == "empty" {
       local.IP = IPv4zero
       if family.network == "tcp6" { local.IP = IPv6zero }
      }
      if mode == "fixed" {
       reserve, err := Listen(family.network, family.address)
       if err != nil { t.Fatal(err) }
       local.Port = reserve.Addr().(*TCPAddr).Port
       if err := reserve.Close(); err != nil { t.Fatal(err) }
      }
      d := Dialer{LocalAddr: local, Timeout: 5*time.Second}
      var c Conn
      var err error
      network, address := family.network, remote.String()
      if mode == "empty" {
       _, port, e := SplitHostPort(address)
       if e != nil { t.Fatal(e) }
       network, address = "tcp", JoinHostPort("", port)
      }
      typedRemote := remote
      if mode == "empty" { typedRemote = &TCPAddr{Port: remote.Port} }
      switch api {
      case "dial": c, err = d.Dial(network, address)
      case "context": c, err = d.DialContext(context.Background(), network, address)
      case "typed": c, err = DialTCP(network, local, typedRemote)
      case "typed-context":
       d.LocalAddr = &TCPAddr{Port: -1}
       c, err = d.DialTCP(context.Background(), network, local.AddrPort(), typedRemote.AddrPort())
      }
      if err != nil { t.Fatal(err) }
      defer c.Close()
      peer, err := listener.Accept()
      if err != nil { t.Fatal(err) }
      defer peer.Close()
      bound := c.LocalAddr().(*TCPAddr)
      if !bound.IP.Equal(remote.IP) || bound.Port == 0 || local.Port != 0 && bound.Port != local.Port { t.Fatalf("local = %v, want %v", bound, local) }
      if peer.RemoteAddr().String() != bound.String() { t.Fatalf("peer address = %v, want %v", peer.RemoteAddr(), bound) }
     })
    }
   }
   for _, invalid := range []struct{name string; port int; errno syscall.Errno}{{"negative", -1, syscall.EINVAL}, {"large", 65536, syscall.EINVAL}, {"occupied", remote.Port, syscall.EADDRINUSE}} {
    apis := []string{"dial", "typed"}
    if invalid.name == "occupied" { apis = append(apis, "typed-context") }
    for _, api := range apis {
    t.Run(invalid.name+"/"+api, func(t *testing.T) {
     local := &TCPAddr{IP: remote.IP, Port: invalid.port}
     d := Dialer{LocalAddr: local, Timeout: 5*time.Second}
     var c Conn
     var err error
     switch api {
     case "dial": c, err = d.Dial(family.network, remote.String())
     case "typed": c, err = DialTCP(family.network, local, remote)
     case "typed-context":
      d.LocalAddr = &TCPAddr{Port: -1}
      c, err = d.DialTCP(context.Background(), family.network, local.AddrPort(), remote.AddrPort())
     }
     if err == nil { c.Close(); t.Fatal("invalid local port succeeded") }
     var op *OpError
     var call *os.SyscallError
     if !errors.Is(err, invalid.errno) || !errors.As(err, &op) || !errors.As(err, &call) { t.Fatalf("error = %T %v, want %v with OpError and SyscallError", err, err, invalid.errno) }
     if op.Op != "dial" || op.Net != family.network || op.Source == nil || op.Source.String() != local.String() || op.Addr == nil || op.Addr.String() != remote.String() || call.Syscall != "bind" { t.Fatalf("error metadata = %+v, syscall = %s", op, call.Syscall) }
    })
    }
   }
  })
 }
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
	t.Log(run(t, dir, "go", "test", "-run", "^TestTCPDial(Families|Refused|LocalBinding|Ports|Canceled)$", "-v", "."))
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
			args := []string{"-test.v", "-test.run=^TestTCPDial(Families|Refused|LocalBinding|Ports|Canceled)$"}
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
	const filter = "^(TestDialLocal|TestDialAddrError|TestDialerLocalAddr|TestCancelAfterDial|TestDialWithNonZeroDeadline)$"
	args := upstreamTestArgs("-test.run=" + filter)
	result := programOutcome(t, netDir, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
	compareUpstream(t, "net-tcp-dial-family", "net", filter, "vm", netDir, result.Text, result.Exit)
	t.Run("aot", func(t *testing.T) {
		result := runUpstreamTarget(t, "aot", out, stats.Binaries[0].Entry, netDir, args)
		compareUpstream(t, "net-tcp-dial-family", "net", filter, "aot", netDir, result.Text, result.Exit)
	})
}
