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
	"strconv"
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
`)
	run(t, dir, "go", "test", "-v", ".")
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true})
	if err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			if target == "vm" {
				run(t, dir, "dart", filepath.Join(out, stats.Binaries[0].Entry), "-test.v")
			} else {
				result := runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, []string{"-test.v"})
				if result.Exit != 0 {
					t.Fatal(result.Text)
				}
			}
		})
	}
}
