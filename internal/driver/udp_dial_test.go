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
 "errors"
 "net"
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
func TestDialArguments(t *testing.T) {
 for _, local := range []*net.UDPAddr{nil, {IP: net.IPv4(127, 0, 0, 1), Port: 1}} {
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
					args = append(args, "-test.run=^TestDialArguments$")
				}
				result := runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
				if result.Exit != 0 {
					t.Fatal(result.Text)
				}
			}
		})
	}
}
