package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestNativeTCPListenerRawConn(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native TCP listener callbacks require Linux or macOS")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/listenerraw\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
	"errors"
	"fmt"
	"net"
	"runtime"
	"syscall"
)

func must(err error) {
	if err != nil { panic(err) }
}

func restricted(raw syscall.RawConn) {
	calls := 0
	read := raw.Read(func(uintptr) bool { calls++; return true })
	write := raw.Write(func(uintptr) bool { calls++; return true })
	fmt.Println("restricted", errors.Is(read, syscall.EINVAL), errors.Is(write, syscall.EINVAL), calls)
}

func controlPanic(raw syscall.RawConn) {
	defer func() { fmt.Println("panic", recover() == "callback") }()
	must(raw.Control(func(uintptr) { panic("callback") }))
}

func held(raw syscall.RawConn, ready chan<- struct{}, release <-chan struct{}, done chan<- error) {
	done <- raw.Control(func(fd uintptr) {
		value, err := syscall.GetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_TYPE)
		must(err)
		if value != syscall.SOCK_STREAM { panic("not a stream") }
		ready <- struct{}{}
		<-release
		value, err = syscall.GetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_TYPE)
		must(err)
		if value != syscall.SOCK_STREAM { panic("descriptor closed during callback") }
	})
}

func main() {
	for _, listener := range []*net.TCPListener{nil, new(net.TCPListener)} {
		raw, err := listener.SyscallConn()
		fmt.Println("invalid", raw == nil, errors.Is(err, syscall.EINVAL))
	}
	for _, network := range []string{"tcp4", "tcp6"} {
		host := "127.0.0.1"
		if network == "tcp6" { host = "::1" }
		listener, err := net.ListenTCP(network, &net.TCPAddr{IP: net.ParseIP(host)})
		must(err)
		raw, err := listener.SyscallConn()
		must(err)
		fmt.Println("network", network)
		must(raw.Control(func(fd uintptr) {
			value, err := syscall.GetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_TYPE)
			must(err)
			fmt.Println("stream", value == syscall.SOCK_STREAM)
		}))
		restricted(raw)
		controlPanic(raw)
		ready := make(chan struct{}, 2)
		first, second := make(chan struct{}), make(chan struct{})
		done := make(chan error, 2)
		go held(raw, ready, first, done)
		go held(raw, ready, second, done)
		<-ready
		<-ready
		started, closed := make(chan struct{}), make(chan error, 1)
		go func() { close(started); closed <- listener.Close() }()
		<-started
		runtime.Gosched()
		select { case <-closed: panic("close passed active callbacks"); default: }
		close(first)
		must(<-done)
		select { case <-closed: panic("close passed the second callback"); default: }
		close(second)
		must(<-done)
		must(<-closed)
		fmt.Println("retained", true)
		called := false
		err = raw.Control(func(uintptr) { called = true })
		var operation *net.OpError
		fmt.Println("closed", errors.Is(err, net.ErrClosed), called, errors.As(err, &operation))
		fmt.Println("metadata", operation.Op, operation.Net, operation.Source == nil, operation.Addr.String() == listener.Addr().String())
		restricted(raw)
		raw, err = listener.SyscallConn()
		must(err)
		err = raw.Control(func(uintptr) { called = true })
		fmt.Println("closed query", raw != nil, errors.Is(err, net.ErrClosed))
	}
}
`)
	want := programOutcome(t, dir, "go", "run", ".")
	if want.Exit != 0 {
		t.Fatalf("native Go failed: %+v", want)
	}
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			var got programResult
			if target == "vm" {
				got = programOutcome(t, dir, "dart", filepath.Join(out, "bin/main.dart"))
			} else {
				got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
			}
			if got != want {
				t.Fatalf("Go:\n%s\nGart (exit %d):\n%s", want.Text, got.Exit, got.Text)
			}
		})
	}
}
