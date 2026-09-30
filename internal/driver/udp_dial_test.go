package driver_test

import (
	"path/filepath"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestUDPDialErrors(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/udpdial\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "dial_test.go"), `package udpdial

import (
 "context"
 "errors"
 "net"
 "net/netip"
 "os"
 "syscall"
 "testing"
)

func TestDialPorts(t *testing.T) {
 for _, network := range []string{"udp", "udp4", "udp6"} {
  ip := net.IPv4(127, 0, 0, 1)
  if network == "udp6" { ip = net.IPv6loopback }
  for _, port := range []int{-1, 65536} {
   for _, mode := range []string{"remote", "local", "both", "occupied"} {
    remote := &net.UDPAddr{IP: ip, Port: port}
    var local *net.UDPAddr
    call, errno := "connect", syscall.EINVAL
    if mode == "local" || mode == "both" {
     local = &net.UDPAddr{IP: ip, Port: port}
     call = "bind"
     if mode == "local" { remote.Port = 9 }
    }
    var server *net.UDPConn
    if mode == "occupied" {
     var err error
     server, err = net.ListenUDP(network, &net.UDPAddr{IP: ip})
     if err != nil { t.Fatal(err) }
     local = server.LocalAddr().(*net.UDPAddr)
     call, errno = "bind", syscall.EADDRINUSE
    }
    conn, err := net.DialUDP(network, local, remote)
    if server != nil { server.Close() }
    if conn != nil { conn.Close(); t.Fatalf("%s %s %d accepted invalid port", network, mode, port) }
    var op *net.OpError
    var se *os.SyscallError
    if !errors.As(err, &op) || op.Op != "dial" || op.Net != network || op.Addr != remote {
     t.Fatalf("incorrect dial metadata: %#v", op)
    }
    if local == nil && op.Source != nil || local != nil && op.Source != local {
     t.Fatalf("incorrect source: %#v", op.Source)
    }
    if !errors.As(err, &se) || se.Syscall != call || !errors.Is(err, errno) {
     t.Fatalf("%s %s %d: %v, want %s %v", network, mode, port, err, call, errno)
    }
   }
  }
 }
}
func TestDialRemoteArguments(t *testing.T) {
 for _, tt := range []struct{network string; ip net.IP; ipv6 bool}{
  {"udp", net.IP{1}, false}, {"udp", net.IP{1,2,3,4,5}, true},
  {"udp4", net.IP{1}, false}, {"udp4", net.IP{1,2,3,4,5}, false}, {"udp4", net.IPv6loopback, false},
  {"udp6", net.IP{1}, true}, {"udp6", net.IP{1,2,3,4,5}, true},
 } {
  ip := net.IPv4(127,0,0,1)
  message := "non-IPv4 address"
  if tt.ipv6 { ip = net.IPv6loopback; message = "non-IPv6 address" }
  for _, mode := range []string{"free", "occupied", "invalid-port"} {
   for _, port := range []int{9,65536} {
    server, err := net.ListenUDP(tt.network, &net.UDPAddr{IP: ip})
    if err != nil { t.Fatal(err) }
    local := server.LocalAddr().(*net.UDPAddr)
    if mode != "occupied" { server.Close() }
    if mode == "invalid-port" { local.Port = -1 }
    remote := &net.UDPAddr{IP: tt.ip, Port: port}
    c, err := net.DialUDP(tt.network, local, remote)
    if c != nil { c.Close(); t.Fatal("invalid remote returned a connection") }
    var op *net.OpError
    if !errors.As(err,&op) || op.Op != "dial" || op.Net != tt.network || op.Source != local || op.Addr != remote { t.Fatalf("operation error: %T %v",err,err) }
    if mode == "free" {
     var address *net.AddrError
     if !errors.As(err,&address) || address.Err != message || address.Addr != tt.ip.String() { t.Fatalf("%s %v port=%d: %T %v, want %s",tt.network,tt.ip,port,err,err,message) }
     rebound, err := net.ListenUDP(tt.network, local)
     if err != nil { t.Fatalf("failed dial retained local port: %v",err) }
     rebound.Close()
    } else {
     want := syscall.EADDRINUSE
     if mode == "invalid-port" { want = syscall.EINVAL }
     var call *os.SyscallError
     if !errors.Is(err,want) || !errors.As(err,&call) || call.Syscall != "bind" { t.Fatalf("%s %v %s port=%d: %v, want bind %v",tt.network,tt.ip,mode,port,err,want) }
    }
    if mode == "occupied" { server.Close() }
   }
  }
 }
}

func TestDialRemoteAutomatic(t *testing.T) {
 for _, tt := range []struct{network string; ip net.IP; message string}{
  {"udp", net.IP{1}, "non-IPv4 address"}, {"udp", net.IP{1,2,3,4,5}, "non-IPv6 address"},
  {"udp4", net.IP{1}, "non-IPv4 address"}, {"udp4", net.IP{1,2,3,4,5}, "non-IPv4 address"},
  {"udp4", net.IPv6loopback, "non-IPv4 address"},
  {"udp6", net.IP{1}, "non-IPv6 address"}, {"udp6", net.IP{1,2,3,4,5}, "non-IPv6 address"},
 } {
  for _, port := range []int{9,65536} {
   remote := &net.UDPAddr{IP: tt.ip, Port: port}
   c, err := net.DialUDP(tt.network, nil, remote)
   if c != nil { c.Close(); t.Fatal("invalid remote returned a connection") }
   var op *net.OpError
   var address *net.AddrError
   if !errors.As(err,&op) || op.Op != "dial" || op.Net != tt.network || op.Source != nil || op.Addr != remote || !errors.As(err,&address) || address.Err != tt.message || address.Addr != tt.ip.String() { t.Fatalf("%s %v port=%d: %T %v, want %s",tt.network,tt.ip,port,err,err,tt.message) }
  }
 }
}

func TestDialArguments(t *testing.T) {
 for _, local := range []*net.UDPAddr{nil, {IP: net.IPv4(127, 0, 0, 1), Port: 1}, {IP: net.IP{1,2,3}, Port: -1}} {
  for _, remote := range []*net.UDPAddr{nil, {IP: net.IPv4(127, 0, 0, 1), Port: 2}} {
   for _, network := range []string{"", "tcp", "udp", "udp4", "udp6"} {
    unknown := network == "" || network == "tcp"
    if !unknown && remote != nil { continue }
    conn, err := net.DialUDP(network, local, remote)
    if conn != nil { conn.Close(); t.Fatal("invalid dial arguments succeeded") }
    var op *net.OpError
    if !errors.As(err, &op) || op.Op != "dial" || op.Net != network {
     t.Fatalf("incorrect dial error: %v", err)
    }
    if local == nil && op.Source != nil || local != nil && op.Source != local {
     t.Fatalf("incorrect source: %#v", op.Source)
    }
    if remote == nil && op.Addr != nil || remote != nil && op.Addr != remote {
     t.Fatalf("incorrect destination: %#v", op.Addr)
    }
    if unknown {
     var cause net.UnknownNetworkError
     if !errors.As(err, &cause) || string(cause) != network { t.Fatalf("incorrect network error: %v", err) }
    } else if op.Err.Error() != "missing address" { t.Fatalf("incorrect missing address error: %v", err) }
   }
  }
 }
}

func TestDialLocalArguments(t *testing.T) {
 for _, tt := range []struct{network string; local, remote net.IP; cause string}{
  {"udp4", net.IP{1}, nil, "non-IPv4 address"},
  {"udp4", net.IP{1,2,3}, net.IPv4(127,0,0,1), "non-IPv4 address"},
  {"udp4", net.IP{1,2,3,4,5}, net.IPv6loopback, "non-IPv4 address"},
  {"udp4", make(net.IP,17), nil, "non-IPv4 address"},
  {"udp4", net.IPv6loopback, net.IPv4(127,0,0,1), "non-IPv4 address"},
  {"udp4", net.IPv6zero, net.IPv6loopback, "non-IPv4 address"},
  {"udp6", net.IP{1}, nil, "non-IPv6 address"},
  {"udp6", net.IP{1,2,3}, net.IPv4(127,0,0,1), "non-IPv6 address"},
  {"udp6", net.IP{1,2,3,4,5}, net.IPv6loopback, "non-IPv6 address"},
  {"udp6", make(net.IP,17), nil, "non-IPv6 address"},
  {"udp", net.IP{1}, nil, "non-IPv4 address"},
  {"udp", net.IP{1,2,3}, net.IPv4(127,0,0,1), "non-IPv4 address"},
  {"udp", net.IP{1}, net.IPv6loopback, "non-IPv6 address"},
  {"udp", net.IP{1,2,3,4,5}, nil, "non-IPv6 address"},
  {"udp", make(net.IP,17), net.IPv4(127,0,0,1), "non-IPv6 address"},
  {"udp", net.IP{1,2,3,4,5}, net.IPv6loopback, "non-IPv6 address"},
 } {
  for _, port := range []int{0, -1} {
   local := &net.UDPAddr{IP: tt.local, Port: port}
   remote := &net.UDPAddr{IP: tt.remote, Port: 9}
   c, err := net.DialUDP(tt.network, local, remote)
   if c != nil { c.Close(); t.Fatal("invalid local address succeeded") }
   var op *net.OpError
   var cause *net.AddrError
   if !errors.As(err, &op) || op.Op != "dial" || op.Net != tt.network || op.Source != local || op.Addr != remote {
    t.Fatalf("incorrect local address metadata: %v", err)
   }
   if !errors.As(err, &cause) || cause.Err != tt.cause || cause.Addr != tt.local.String() {
    t.Fatalf("%s %v -> %v: %v, want %s", tt.network, local, remote, err, tt.cause)
   }
  }
 }
}

func TestDialerArguments(t *testing.T) {
 addresses := []netip.AddrPort{{}, netip.MustParseAddrPort("127.0.0.1:1"), netip.MustParseAddrPort("[::1%zone]:2")}
 for _, canceled := range []bool{false, true} {
  ctx, cancel := context.WithCancel(context.Background())
  if canceled { cancel() }
  for _, local := range addresses {
   for _, remote := range addresses {
    for _, network := range []string{"", "tcp"} {
     d := net.Dialer{LocalAddr: &net.TCPAddr{}}
     c, err := d.DialUDP(ctx, network, local, remote)
     if c != nil { c.Close(); t.Fatal("unknown dialer network succeeded") }
     var op *net.OpError
     var cause net.UnknownNetworkError
     if !errors.As(err, &op) || op.Op != "dial" || op.Net != network || !errors.As(err, &cause) || string(cause) != network {
      t.Fatalf("incorrect dialer error: %v", err)
     }
     source, sourceOK := op.Source.(*net.UDPAddr)
     dest, destOK := op.Addr.(*net.UDPAddr)
     if !sourceOK || !destOK || source.AddrPort() != local || dest.AddrPort() != remote {
      t.Fatalf("incorrect dialer addresses: %#v", op)
     }
    }
   }
  }
  cancel()
 }
 for _, network := range []string{"udp", "tcp"} {
  func() {
   defer func() { if p := recover(); p != "nil context" { t.Fatalf("nil context panic: %v", p) } }()
   var d net.Dialer
   d.DialUDP(nil, network, netip.AddrPort{}, netip.AddrPort{})
  }()
  func() {
   defer func() { if recover() == nil { t.Fatal("nil dialer did not panic") } }()
   var d *net.Dialer
   d.DialUDP(context.Background(), network, netip.AddrPort{}, netip.AddrPort{})
  }()
 }
}

`)
	run(t, dir, "go", "test", "-v", ".")
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true})
	if err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			if target == "vm" {
				run(t, dir, "dart", filepath.Join(out, stats.Binaries[0].Entry), "-test.v")
			} else {
				args := []string{"-test.v"}
				if target == "js" || target == "wasm" {
					args = append(args, "-test.run=^(TestDialArguments|TestDialLocalArguments|TestDialerArguments|TestDialRemoteAutomatic)$")
				}
				result := runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
				if result.Exit != 0 {
					t.Fatal(result.Text)
				}
			}
		})
	}
}
