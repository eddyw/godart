package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestStreamConnContracts(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("stream contract validation requires macOS or Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/tcpcontract\n\ngo 1.27\n")
	if e := os.Mkdir(filepath.Join(dir, "nettest"), 0755); e != nil {
		t.Fatal(e)
	}
	source, e := os.ReadFile(filepath.Join(runtime.GOROOT(), "src/vendor/golang.org/x/net/nettest/conntest.go"))
	if e != nil {
		t.Fatal(e)
	}
	writeFile(t, filepath.Join(dir, "nettest/conntest.go"), string(source))
	writeFile(t, filepath.Join(dir, "nettest/constants.go"), selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/vendor/golang.org/x/net/nettest/nettest.go"), []string{"aLongTimeAgo", "neverTimeout"}))
	license, e := os.ReadFile(filepath.Join(runtime.GOROOT(), "src/vendor/golang.org/x/net/LICENSE"))
	if e != nil {
		t.Fatal(e)
	}
	writeFile(t, filepath.Join(dir, "nettest/LICENSE"), string(license))
	writeFile(t, filepath.Join(dir, "contract_test.go"), `package tcpcontract

import (
	"errors"
	"example.com/tcpcontract/nettest"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStreamContract(t *testing.T) {
	for _, network := range []string{"tcp", "unix"} {
		t.Run(network, func(t *testing.T) {
			nettest.TestConn(t, func() (a, b net.Conn, stop func(), err error) {
				address := "127.0.0.1:0"
				if network == "unix" {
					dir, e := os.MkdirTemp("", "gart-conn")
					if e != nil {
						return nil, nil, nil, e
					}
					t.Cleanup(func() { os.RemoveAll(dir) })
					address = filepath.Join(dir, "s")
				}
				l, err := net.Listen(network, address)
				if err != nil {
					return nil, nil, nil, err
				}
				a, err = net.Dial(network, l.Addr().String())
				if err != nil {
					l.Close()
					return nil, nil, nil, err
				}
				b, err = l.Accept()
				if err != nil {
					a.Close()
					l.Close()
					return nil, nil, nil, err
				}
				stop = func() { a.Close(); b.Close(); l.Close() }
				return
			})
		})
	}
}
func TestTCPPartialWrite(t *testing.T) {
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	defer l.Close()
	c, e := net.Dial("tcp", l.Addr().String())
	if e != nil {
		t.Fatal(e)
	}
	defer c.Close()
	p, e := l.Accept()
	if e != nil {
		t.Fatal(e)
	}
	defer p.Close()
	if e := c.(*net.TCPConn).SetWriteBuffer(4096); e != nil {
		t.Fatal(e)
	}
	if e := p.(*net.TCPConn).SetReadBuffer(4096); e != nil {
		t.Fatal(e)
	}
	payload := make([]byte, 4<<20)
	type result struct {
		n   int
		err error
	}
	done := make(chan result, 1)
	go func() { n, e := c.Write(payload); done <- result{n, e} }()
	time.Sleep(100 * time.Millisecond)
	c.SetWriteDeadline(time.Unix(1, 0))
	r := <-done
	if r.n <= 0 || r.n >= len(payload) || !errors.Is(r.err, os.ErrDeadlineExceeded) {
		t.Fatalf("write = %d %v, want partial count and deadline exceeded", r.n, r.err)
	}
	p.SetReadDeadline(time.Now().Add(5 * time.Second))
	if n, e := io.CopyN(io.Discard, p, int64(r.n)); n != int64(r.n) || e != nil {
		t.Fatalf("drain = %d %v, want %d bytes", n, e, r.n)
	}
	p.SetReadDeadline(time.Now().Add(30 * time.Millisecond))
	b := make([]byte, 5)
	if n, e := p.Read(b); n != 0 || !errors.Is(e, os.ErrDeadlineExceeded) {
		t.Fatalf("write continued after timeout: read %d %v", n, e)
	}
	c.SetWriteDeadline(time.Now().Add(time.Second))
	p.SetReadDeadline(time.Now().Add(time.Second))
	if n, e := c.Write([]byte("after")); n != 5 || e != nil {
		t.Fatalf("next write = %d %v", n, e)
	}
	if _, e := io.ReadFull(p, b); e != nil || string(b) != "after" {
		t.Fatalf("next read = %q %v", b, e)
	}
}

func TestTCPQueuedWriteClose(t *testing.T) {
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	defer l.Close()
	c, e := net.Dial("tcp", l.Addr().String())
	if e != nil {
		t.Fatal(e)
	}
	defer c.Close()
	p, e := l.Accept()
	if e != nil {
		t.Fatal(e)
	}
	defer p.Close()
	if e := c.(*net.TCPConn).SetWriteBuffer(4096); e != nil {
		t.Fatal(e)
	}
	if e := p.(*net.TCPConn).SetReadBuffer(4096); e != nil {
		t.Fatal(e)
	}
	c.SetWriteDeadline(time.Now().Add(3 * time.Second))
	payload := make([]byte, 4<<20)
	ready := make(chan bool)
	done := make(chan error, 3)
	for i := 0; i < 3; i++ {
		go func() { ready <- true; _, e := c.Write(payload); done <- e }()
	}
	<-ready
	<-ready
	<-ready
	time.Sleep(100 * time.Millisecond)
	if e := c.Close(); e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 3; i++ {
		if e := <-done; !errors.Is(e, net.ErrClosed) {
			t.Fatalf("queued write: %v, want closed", e)
		}
	}
}

func TestTCPQueuedEmptyWrite(t *testing.T) {
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	defer l.Close()
	c, e := net.Dial("tcp", l.Addr().String())
	if e != nil {
		t.Fatal(e)
	}
	defer c.Close()
	p, e := l.Accept()
	if e != nil {
		t.Fatal(e)
	}
	defer p.Close()
	if e := c.(*net.TCPConn).SetWriteBuffer(4096); e != nil {
		t.Fatal(e)
	}
	if e := p.(*net.TCPConn).SetReadBuffer(4096); e != nil {
		t.Fatal(e)
	}
	c.SetWriteDeadline(time.Now().Add(3 * time.Second))
	done := make(chan error, 1)
	empty := make(chan error, 1)
	go func() { _, e := c.Write(make([]byte, 4<<20)); done <- e }()
	time.Sleep(100 * time.Millisecond)
	go func() { _, e := c.Write(nil); empty <- e }()
	select {
	case e := <-empty:
		t.Fatalf("empty write bypassed blocked write: %v", e)
	case <-time.After(30 * time.Millisecond):
	}
	if e := c.Close(); e != nil {
		t.Fatal(e)
	}
	for _, ch := range []chan error{done, empty} {
		if e := <-ch; !errors.Is(e, net.ErrClosed) {
			t.Fatalf("queued write: %v, want closed", e)
		}
	}
}
`)
	const filter = "^(TestStreamContract|TestTCPPartialWrite|TestTCPQueuedWriteClose|TestTCPQueuedEmptyWrite)$"
	t.Log(run(t, dir, "go", "test", "-v", "-run", filter, "."))
	stats, e := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true})
	if e != nil {
		t.Fatal(e)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			args := []string{"-test.v", "-test.run=" + filter}
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
}
