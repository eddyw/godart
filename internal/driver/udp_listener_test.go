package driver_test

import (
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestUDPListenerPortOwnership(t *testing.T) {
	probe, err := net.ListenUDP("udp6", &net.UDPAddr{IP: net.IPv6loopback})
	if err != nil {
		t.Skipf("IPv6 loopback is unavailable: %v", err)
	}
	probe.Close()
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/udplistener\n\ngo 1.27\n")
	source := selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/net/listen_test.go"), []string{"udpListenerTests", "checkSecondListener"})
	source = strings.Replace(source, "package net\n", "package udplistener\nimport . \"net\"\n", 1)
	writeFile(t, filepath.Join(dir, "upstream_test.go"), source)
	license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
	writeFile(t, filepath.Join(dir, "ownership_test.go"), `package udplistener

import (
	"context"
	"errors"
	. "net"
	"os"
	"strconv"
	"syscall"
	"testing"
	"time"
)

func TestPortOwnership(t *testing.T) {
	for _, api := range []string{"ListenUDP", "ListenPacket", "ListenConfig"} {
		listen := func(network, address string) (PacketConn, error) {
			switch api {
			case "ListenPacket":
				return ListenPacket(network, address)
			case "ListenConfig":
				var lc ListenConfig
				return lc.ListenPacket(context.Background(), network, address)
			default:
				addr, err := ResolveUDPAddr(network, address)
				if err != nil {
					return nil, err
				}
				conn, err := ListenUDP(network, addr)
				if err != nil {
					return nil, err
				}
				return conn, nil
			}
		}
		for _, tt := range udpListenerTests {
			t.Run(api+"/"+tt.network+"/"+tt.address, func(t *testing.T) {
				first, err := listen(tt.network, JoinHostPort(tt.address, "0"))
				if err != nil {
					t.Fatal(err)
				}
				defer first.Close()
				taken := first.LocalAddr().(*UDPAddr)
				if taken.Port == 0 {
					t.Fatal("listener has no port")
				}
				if tt.network != "udp" && (taken.IP.To4() != nil) != (tt.network == "udp4") {
					t.Fatalf("incorrect address family: %v", taken)
				}
				second, err := listen(tt.network, taken.String())
				if err == nil {
					second.Close()
				}
				if err := checkSecondListener(tt.network, tt.address, err); err != nil {
					t.Fatal(err)
				}
				var op *OpError
				var call *os.SyscallError
				if second != nil || !errors.As(err, &op) || op.Op != "listen" || op.Net != tt.network || op.Source != nil {
					t.Fatalf("incorrect bind result: %v %#v", second, op)
				}
				addr, ok := op.Addr.(*UDPAddr)
				if !ok || addr.Port != taken.Port || !(addr.IP.Equal(taken.IP) || addr.IP.IsUnspecified() && taken.IP.IsUnspecified()) {
					t.Fatalf("incorrect bind address: %v, want %v", op.Addr, taken)
				}
				if !errors.As(err, &call) || call.Syscall != "bind" || !errors.Is(err, syscall.EADDRINUSE) {
					t.Fatalf("incorrect bind error: %v", err)
				}
				peer := taken.IP.String()
				if taken.IP.IsUnspecified() {
					if taken.IP.To4() != nil {
						peer = "127.0.0.1"
					} else {
						peer = "::1"
					}
				}
				client, err := Dial(tt.network, JoinHostPort(peer, strconv.Itoa(taken.Port)))
				if err != nil {
					t.Fatal(err)
				}
				defer client.Close()
				if err := client.SetWriteDeadline(time.Now().Add(time.Second)); err != nil {
					t.Fatal(err)
				}
				if _, err := client.Write([]byte("hello")); err != nil {
					t.Fatal(err)
				}
				if err := first.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
					t.Fatal(err)
				}
				b := make([]byte, 16)
				n, from, err := first.ReadFrom(b)
				if err != nil || string(b[:n]) != "hello" || from.String() != client.LocalAddr().String() {
					t.Fatalf("datagram = %q %v %v", b[:n], from, err)
				}
				if err := first.Close(); err != nil {
					t.Fatal(err)
				}
				next, err := listen(tt.network, taken.String())
				if err != nil {
					t.Fatal(err)
				}
				next.Close()
			})
		}
	}
}
func TestPacketAddressErrors(t *testing.T) {
	for _, tt := range []struct{ network, address string }{
		{"udp4", "[::1]:0"}, {"udp6", "127.0.0.1:0"}, {"udp6", "[::ffff:127.0.0.1]:0"},
		{"udp", "127.0.0.1:-1"}, {"udp", "127.0.0.1:65536"}, {"udp", "127.0.0.1"},
	} {
		c, err := ListenPacket(tt.network, tt.address)
		if c != nil {
			c.Close()
			t.Fatalf("non-nil connection for %v", tt)
		}
		var op *OpError
		var addr *AddrError
		if !errors.As(err, &op) || op.Op != "listen" || op.Net != tt.network || !errors.As(err, &addr) {
			t.Fatalf("incorrect error for %v: %T %v", tt, err, err)
		}
	}
	for _, network := range []string{"", "unknown"} {
		c, err := ListenPacket(network, "127.0.0.1:0")
		if c != nil {
			c.Close()
			t.Fatalf("non-nil connection for %q", network)
		}
		var op *OpError
		var unknown UnknownNetworkError
		if !errors.As(err, &op) || op.Op != "listen" || op.Net != network || !errors.As(err, &unknown) {
			t.Fatalf("incorrect network error: %T %v", err, err)
		}
	}
}
func TestTypedListenerErrors(t *testing.T) {
	for _, network := range []string{"udp", "udp4", "udp6"} {
		for _, ip := range []IP{{1}, {1, 2, 3}, {1, 2, 3, 4, 5}, make(IP, 17), IPv6loopback, IPv4(127, 0, 0, 1)} {
			for _, port := range []int{0, -1, 65536} {
				invalidIP := ip.To16() == nil || network == "udp4" && ip.To4() == nil
				if !invalidIP && (port == 0 || network == "udp6" && ip.To4() != nil) {
					continue
				}
				addr := &UDPAddr{IP: ip, Port: port}
				c, err := ListenUDP(network, addr)
				if c != nil {
					c.Close()
					t.Fatalf("accepted %s %v", network, addr)
				}
				var op *OpError
				if !errors.As(err, &op) || op.Op != "listen" || op.Net != network || op.Source != nil || op.Addr != addr {
					t.Fatalf("incorrect operation for %s %v: %#v", network, addr, op)
				}
				if invalidIP {
					var bad *AddrError
					want := "non-IPv4 address"
					if network == "udp6" || network == "udp" && len(ip) > IPv4len {
						want = "non-IPv6 address"
					}
					if !errors.As(err, &bad) || bad.Err != want || bad.Addr != ip.String() {
						t.Fatalf("incorrect address error: %v", err)
					}
				} else {
					var call *os.SyscallError
					if !errors.As(err, &call) || call.Syscall != "bind" || !errors.Is(err, syscall.EINVAL) {
						t.Fatalf("incorrect port error: %v", err)
					}
				}
			}
		}
	}
	for _, addr := range []*UDPAddr{nil, {IP: IPv4(127, 0, 0, 1)}} {
		c, err := ListenUDP("unknown", addr)
		if c != nil {
			c.Close()
			t.Fatal("accepted unknown network")
		}
		var op *OpError
		var unknown UnknownNetworkError
		if !errors.As(err, &op) || op.Op != "listen" || op.Net != "unknown" || !errors.As(err, &unknown) {
			t.Fatalf("incorrect unknown network error: %v", err)
		}
		if addr == nil && op.Addr != nil || addr != nil && op.Addr != addr {
			t.Fatalf("incorrect error address: %#v", op.Addr)
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
					args = append(args, "-test.run=^(TestPacketAddressErrors|TestTypedListenerErrors)$")
				}
				result := runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
				if result.Exit != 0 {
					t.Fatal(result.Text)
				}
			}
		})
	}
}
