package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNativeInetAddressCalls(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("Native Internet address calls require Linux or macOS")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/inetaddress\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
	"bytes"
	"fmt"
	"net"
	"syscall"
	"time"
)

func result(label string, err error) {
	code, typed := err.(syscall.Errno)
	fmt.Println(label, code, typed)
}

func receive(label string, receiver *net.UDPConn, want []byte) *net.UDPAddr {
	if err := receiver.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		panic(err)
	}
	buffer := make([]byte, 32)
	n, peer, err := receiver.ReadFromUDP(buffer)
	if err != nil {
		panic(err)
	}
	if !bytes.Equal(buffer[:n], want) {
		panic("packet bytes differ")
	}
	fmt.Println(label, n, buffer[:n], peer.Port > 0, peer.IP.IsLoopback())
	return peer
}

func main() {
	for _, network := range []string{"udp4", "udp6"} {
		ip := net.ParseIP("127.0.0.1")
		family := syscall.AF_INET
		if network == "udp6" {
			ip = net.ParseIP("::1")
			family = syscall.AF_INET6
		}
		receiver, err := net.ListenUDP(network, &net.UDPAddr{IP: ip})
		if err != nil {
			panic(err)
		}
		port := receiver.LocalAddr().(*net.UDPAddr).Port
		fd, err := syscall.Socket(family, syscall.SOCK_DGRAM, 0)
		if err != nil {
			panic(err)
		}
		if err := syscall.SetNonblock(fd, true); err != nil {
			panic(err)
		}
		var local, destination syscall.Sockaddr
		if network == "udp4" {
			local = &syscall.SockaddrInet4{Addr: [4]byte{127, 0, 0, 1}}
			destination = &syscall.SockaddrInet4{Port: port, Addr: [4]byte{127, 0, 0, 1}}
		} else {
			local = &syscall.SockaddrInet6{Addr: [16]byte{15: 1}}
			destination = &syscall.SockaddrInet6{Port: port, Addr: [16]byte{15: 1}}
		}
		fmt.Println("family", network)
		result("bind", syscall.Bind(fd, local))
		result("sendto", syscall.Sendto(fd, []byte{0, 128, 255}, 0, destination))
		receive("explicit packet", receiver, []byte{0, 128, 255})
		result("empty sendto", syscall.Sendto(fd, nil, 0, destination))
		receive("empty packet", receiver, nil)
		duplicate, err := syscall.Dup(fd)
		if err != nil {
			panic(err)
		}
		result("connect duplicate", syscall.Connect(duplicate, destination))
		n, err := syscall.Write(fd, []byte{42, 0})
		fmt.Println("connected write", n)
		result("write", err)
		peer := receive("connected packet", receiver, []byte{42, 0})
		result("nil sendto", syscall.Sendto(fd, []byte{255, 0}, 0, nil))
		receive("nil packet", receiver, []byte{255, 0})
		if err := syscall.SetNonblock(fd, false); err != nil { panic(err) }
		n, err = receiver.WriteToUDP([]byte{128, 0, 255}, peer)
		if err != nil || n != 3 {
			panic("response write")
		}
		buffer := make([]byte, 16)
		n, err = syscall.Read(fd, buffer)
		if err != nil {
			panic(err)
		}
		if !bytes.Equal(buffer[:n], []byte{128, 0, 255}) {
			panic("response bytes differ")
		}
		fmt.Println("response", n, buffer[:n])
		for _, invalid := range []int{-1, 65536} {
			var address syscall.Sockaddr
			if network == "udp4" {
				address = &syscall.SockaddrInet4{Port: invalid}
			} else {
				address = &syscall.SockaddrInet6{Port: invalid}
			}
			result("invalid bind", syscall.Bind(-1, address))
			result("invalid connect", syscall.Connect(-1, address))
			result("invalid sendto", syscall.Sendto(-1, nil, 0, address))
		}
		result("valid bad fd", syscall.Connect(-1, destination))
		result("duplicate close", syscall.Close(duplicate))
		typ, err := syscall.GetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_TYPE)
		fmt.Println("owned", typ == syscall.SOCK_DGRAM)
		result("option", err)
		result("close", syscall.Close(fd))
		if err := receiver.Close(); err != nil {
			panic(err)
		}
	}
}
`)
	want := runProgram(t, dir, "go", "run", ".")
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			var got programResult
			if target == "vm" {
				got = upstreamOutcome(t, dir, nil, runLimit, "dart", filepath.Join(out, "bin/main.dart"))
			} else {
				got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
			}
			if got.Exit != 0 || got.Text != want {
				t.Fatalf("%s: exit %d\nGo:\n%s\nGart:\n%s", target, got.Exit, want, got.Text)
			}
		})
	}
}
