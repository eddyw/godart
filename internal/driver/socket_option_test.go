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

func TestSocketOptionErrors(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("socket option validation requires macOS or Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/socketset\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "contract_test.go"), `package socketset

import (
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestLingerNil(t *testing.T) {
 defer func() { if recover() == nil { t.Fatal("nil receiver did not panic") } }()
 (*net.TCPConn)(nil).SetLinger(0)
}
func TestLingerClose(t *testing.T) {
 for _, address := range []string{"127.0.0.1:0", "[::1]:0"} {
  for _, seconds := range []int{-1, 0, 1} {
   listener, err := net.Listen("tcp", address)
   if err != nil { t.Fatal(err) }
   client, err := net.Dial("tcp", listener.Addr().String())
   if err != nil { listener.Close(); t.Fatal(err) }
   peer, err := listener.Accept()
   listener.Close()
   if err != nil { client.Close(); t.Fatal(err) }
   if err := client.SetDeadline(time.Now().Add(2*time.Second)); err != nil { t.Fatal(err) }
   if err := peer.SetDeadline(time.Now().Add(2*time.Second)); err != nil { t.Fatal(err) }
   if err := peer.(*net.TCPConn).SetLinger(seconds); err != nil { t.Fatal(err) }
   if _, err := peer.Write([]byte("ready")); err != nil { t.Fatal(err) }
   b := make([]byte,5)
   if _, err := io.ReadFull(client,b); err != nil { t.Fatal(err) }
   if string(b) != "ready" { t.Fatalf("payload %q", b) }
   if err := peer.Close(); err != nil { t.Fatal(err) }
   n, err := client.Read(b)
   client.Close()
   want := io.EOF
   if seconds == 0 { want = syscall.ECONNRESET }
   if n != 0 || !errors.Is(err,want) { t.Fatalf("%s linger=%d: read=%d %v, want 0 %v", address, seconds, n, err, want) }
  }
 }
}

func TestSocketSet(t *testing.T) {
	for _, network := range []string{"tcp", "udp", "udp-listen", "unix"} {
		t.Run(network, func(t *testing.T) {
			var c net.Conn
			if network == "tcp" || network == "unix" {
				address := "127.0.0.1:0"
				if network == "unix" {
					dir, e := os.MkdirTemp("", "gart-set")
					if e != nil {
						t.Fatal(e)
					}
					defer os.RemoveAll(dir)
					address = filepath.Join(dir, "s")
				}
				l, e := net.Listen(network, address)
				if e != nil {
					t.Fatal(e)
				}
				defer l.Close()
				c, e = net.Dial(network, l.Addr().String())
				if e != nil {
					t.Fatal(e)
				}
				p, e := l.Accept()
				if e != nil {
					t.Fatal(e)
				}
				defer p.Close()
			} else if network == "udp-listen" {
				p, e := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
				if e != nil {
					t.Fatal(e)
				}
				c = p
			} else {
				p, e := net.ListenPacket("udp", "127.0.0.1:0")
				if e != nil {
					t.Fatal(e)
				}
				defer p.Close()
				c, e = net.Dial("udp", p.LocalAddr().String())
				if e != nil {
					t.Fatal(e)
				}
			}
			local, remote := c.LocalAddr(), c.RemoteAddr()
			if tcp, ok := c.(*net.TCPConn); ok {
				for _, option := range []struct {
					name string
					call func() error
				}{
					{"idle", func() error { return tcp.SetKeepAlivePeriod(365 * 24 * time.Hour) }},
					{"interval", func() error { return tcp.SetKeepAliveConfig(net.KeepAliveConfig{Enable: true, Idle: -1, Interval: 365 * 24 * time.Hour, Count: -1}) }},
				} {
					t.Run("open-"+option.name, func(t *testing.T) {
						err := option.call()
						var op *net.OpError
						var call *os.SyscallError
						if !errors.Is(err, syscall.EINVAL) || !errors.As(err, &op) || !errors.As(err, &call) {
							t.Fatalf("error = %T %v, want OpError and SyscallError with EINVAL", err, err)
						}
						if op.Op != "set" || op.Net != "tcp" || call.Syscall != "setsockopt" {
							t.Fatalf("operation = %s %s %s", op.Op, op.Net, call.Syscall)
						}
						if op.Source == nil || op.Addr == nil || op.Source.String() != local.String() || op.Addr.String() != remote.String() {
							t.Fatalf("addresses = %v %v, want %v %v", op.Source, op.Addr, local, remote)
						}
					})
				}
				for _, seconds := range []int{-1, 0, 1, 10, -1} {
					if err := tcp.SetLinger(seconds); err != nil { t.Fatalf("SetLinger(%d): %v", seconds, err) }
				}
				if err := tcp.SetKeepAliveConfig(net.KeepAliveConfig{Enable: true}); err != nil {
					t.Fatal(err)
				}
			}
			if e := c.Close(); e != nil {
				t.Fatal(e)
			}
			type method struct {
				name         string
				call         func() error
				source, addr net.Addr
			}
			methods := []method{
				{"deadline", func() error { return c.SetDeadline(time.Time{}) }, nil, local},
				{"read-deadline", func() error { return c.SetReadDeadline(time.Time{}) }, nil, local},
				{"write-deadline", func() error { return c.SetWriteDeadline(time.Time{}) }, nil, local},
			}
			buffers := c.(interface {
				SetReadBuffer(int) error
				SetWriteBuffer(int) error
			})
			methods = append(methods, method{"read-buffer", func() error { return buffers.SetReadBuffer(4096) }, nil, local}, method{"write-buffer", func() error { return buffers.SetWriteBuffer(4096) }, nil, local})
			if c, ok := c.(*net.TCPConn); ok {
				methods = append(methods,
					method{"linger", func() error { return c.SetLinger(0) }, local, remote},
					method{"nodelay", func() error { return c.SetNoDelay(true) }, local, remote},
					method{"keepalive", func() error { return c.SetKeepAlive(true) }, local, remote},
					method{"keepalive-period", func() error { return c.SetKeepAlivePeriod(time.Second) }, local, remote},
					method{"keepalive-config", func() error { return c.SetKeepAliveConfig(net.KeepAliveConfig{Enable: true}) }, local, remote})
			}
			for _, m := range methods {
				t.Run(m.name, func(t *testing.T) {
					e := m.call()
					var op *net.OpError
					if !errors.Is(e, net.ErrClosed) || !errors.As(e, &op) {
						t.Fatalf("error = %T %v, want OpError wrapping ErrClosed", e, e)
					}
					if op.Op != "set" || op.Net != local.Network() {
						t.Errorf("operation = %q %q, want set %s", op.Op, op.Net, local.Network())
					}
					if (op.Source == nil) != (m.source == nil) || m.source != nil && op.Source.String() != m.source.String() {
						t.Errorf("source = %v, want %v", op.Source, m.source)
					}
					if op.Addr == nil || op.Addr.String() != m.addr.String() {
						t.Errorf("address = %v, want %v", op.Addr, m.addr)
					}
				})
			}
		})
	}
}
`)
	source := selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/net/net_test.go"), []string{"TestNotTemporaryRead"})
	source = strings.Replace(source, "package net\n", "package socketset\nimport . \"net\"\n", 1)
	writeFile(t, filepath.Join(dir, "upstream_test.go"), source)
	license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
	writeFile(t, filepath.Join(dir, "listener_test.go"), `package socketset
import ("net"; "testing")
func newLocalListener(t testing.TB, network string) net.Listener {
 t.Helper()
 listener, err := net.Listen(network, "127.0.0.1:0")
 if err != nil { t.Fatal(err) }
 return listener
}
`)

	t.Log(run(t, dir, "go", "test", "-short="+strconv.FormatBool(testing.Short()), "-v", "."))
	stats, e := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true})
	if e != nil {
		t.Fatal(e)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			args := []string{"-test.v", "-test.short=" + strconv.FormatBool(testing.Short())}
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
			var original programResult
			if target == "vm" {
				original = programOutcome(t, dir, "dart", filepath.Join(out, stats.Binaries[0].Entry), "-test.v", "-test.short="+strconv.FormatBool(testing.Short()), "-test.run=^TestNotTemporaryRead$")
			} else {
				original = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, []string{"-test.v", "-test.short=" + strconv.FormatBool(testing.Short()), "-test.run=^TestNotTemporaryRead$"})
			}
			compareUpstream(t, "net-linger", "net", "^TestNotTemporaryRead$", target, dir, original.Text, original.Exit)
		})
	}
}
