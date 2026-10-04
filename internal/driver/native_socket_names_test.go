package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNativeSocketNames(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("Native socket name calls require Linux or macOS")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/socketnames\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
	"fmt"
	"net"
	"os"
	"syscall"
)

func result(label string, address syscall.Sockaddr, err error) {
	code, typed := err.(syscall.Errno)
	fmt.Println(label, address == nil, code, typed)
}

func same(left, right syscall.Sockaddr) bool {
	switch value := left.(type) {
	case *syscall.SockaddrInet4:
		other, ok := right.(*syscall.SockaddrInet4)
		return ok && value.Port == other.Port && value.Addr == other.Addr
	case *syscall.SockaddrInet6:
		other, ok := right.(*syscall.SockaddrInet6)
		return ok && value.Port == other.Port && value.Addr == other.Addr && value.ZoneId == other.ZoneId
	case *syscall.SockaddrUnix:
		other, ok := right.(*syscall.SockaddrUnix)
		return ok && value.Name == other.Name
	}
	return false
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
		var local, peer syscall.Sockaddr
		if network == "udp4" {
			local = &syscall.SockaddrInet4{Addr: [4]byte{127, 0, 0, 1}}
			peer = &syscall.SockaddrInet4{Port: port, Addr: [4]byte{127, 0, 0, 1}}
		} else {
			local = &syscall.SockaddrInet6{Addr: [16]byte{15: 1}}
			peer = &syscall.SockaddrInet6{Port: port, Addr: [16]byte{15: 1}}
		}
		fmt.Println("family", network)
		address, err := syscall.Getpeername(fd)
		result("unconnected peer", address, err)
		if err := syscall.Bind(fd, local); err != nil {
			panic(err)
		}
		address, err = syscall.Getsockname(fd)
		result("local", address, err)
		switch value := address.(type) {
		case *syscall.SockaddrInet4:
			fmt.Println("actual local", value.Port > 0, value.Addr == [4]byte{127, 0, 0, 1})
		case *syscall.SockaddrInet6:
			fmt.Println("actual local", value.Port > 0, value.Addr == [16]byte{15: 1}, value.ZoneId)
		default:
			panic("local address type")
		}
		if err := syscall.Connect(fd, peer); err != nil {
			panic(err)
		}
		got, err := syscall.Getpeername(fd)
		result("peer", got, err)
		fmt.Println("actual peer", same(got, peer))
		duplicate, err := syscall.Dup(fd)
		if err != nil {
			panic(err)
		}
		got, err = syscall.Getsockname(duplicate)
		result("duplicate local", got, err)
		fmt.Println("same local", same(got, address))
		got, err = syscall.Getpeername(duplicate)
		result("duplicate peer", got, err)
		fmt.Println("same peer", same(got, peer))
		if err := syscall.Close(duplicate); err != nil {
			panic(err)
		}
		got, err = syscall.Getpeername(fd)
		result("owned peer", got, err)
		if err := syscall.Close(fd); err != nil {
			panic(err)
		}
		if err := receiver.Close(); err != nil {
			panic(err)
		}
	}
	pair, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_DGRAM, 0)
	if err != nil {
		panic(err)
	}
	for _, fd := range pair {
		local, err := syscall.Getsockname(fd)
		result("unnamed local", local, err)
		fmt.Printf("actual unnamed local %q\n", local.(*syscall.SockaddrUnix).Name)
		peer, err := syscall.Getpeername(fd)
		result("unnamed peer", peer, err)
		fmt.Printf("actual unnamed peer %q\n", peer.(*syscall.SockaddrUnix).Name)
	}
	for _, fd := range pair { if err := syscall.Close(fd); err != nil { panic(err) } }
	dir, err := os.MkdirTemp("", "raw-socket-names")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	if err := os.Chdir(dir); err != nil {
		panic(err)
	}
	server, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_DGRAM, 0)
	if err != nil {
		panic(err)
	}
	client, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_DGRAM, 0)
	if err != nil {
		panic(err)
	}
	destination, source := &syscall.SockaddrUnix{Name: "server"}, &syscall.SockaddrUnix{Name: "client"}
	if err := syscall.Bind(server, destination); err != nil {
		panic(err)
	}
	if err := syscall.Bind(client, source); err != nil {
		panic(err)
	}
	if err := syscall.Connect(client, destination); err != nil {
		panic(err)
	}
	local, err := syscall.Getsockname(client)
	result("Unix local", local, err)
	fmt.Println("actual Unix local", same(local, source))
	peer, err := syscall.Getpeername(client)
	result("Unix peer", peer, err)
	fmt.Println("actual Unix peer", same(peer, destination))
	local.(*syscall.SockaddrUnix).Name = "changed"
	again, err := syscall.Getsockname(client)
	result("independent local", again, err)
	fmt.Println("independent", same(again, source))
	for _, fd := range []int{client, server} {
		if err := syscall.Close(fd); err != nil {
			panic(err)
		}
	}
	for _, getter := range []func(int) (syscall.Sockaddr, error){syscall.Getsockname, syscall.Getpeername} {
		address, err := getter(-1)
		result("bad fd", address, err)
	}
	file, err := os.Create("regular")
	if err != nil {
		panic(err)
	}
	for _, getter := range []func(int) (syscall.Sockaddr, error){syscall.Getsockname, syscall.Getpeername} {
		address, err := getter(int(file.Fd()))
		result("regular fd", address, err)
	}
	if err := file.Close(); err != nil {
		panic(err)
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
