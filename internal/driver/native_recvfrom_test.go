package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNativeSocketRecvfrom(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("Native socket receive calls require Linux or macOS")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/recvfrom\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
	"fmt"
	"os"
	"syscall"
)

func result(label string, n int, address syscall.Sockaddr, err error) {
	code, typed := err.(syscall.Errno)
	fmt.Println(label, n, address == nil, code, typed)
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

func socket(family int, address syscall.Sockaddr) (int, syscall.Sockaddr) {
	fd, err := syscall.Socket(family, syscall.SOCK_DGRAM, 0)
	if err != nil {
		panic(err)
	}
	if err := syscall.Bind(fd, address); err != nil {
		panic(err)
	}
	actual, err := syscall.Getsockname(fd)
	if err != nil {
		panic(err)
	}
	return fd, actual
}

func main() {
	for _, family := range []int{syscall.AF_INET, syscall.AF_INET6} {
		var local syscall.Sockaddr
		if family == syscall.AF_INET {
			local = &syscall.SockaddrInet4{Addr: [4]byte{127, 0, 0, 1}}
		} else {
			local = &syscall.SockaddrInet6{Addr: [16]byte{15: 1}}
		}
		receiver, destination := socket(family, local)
		sender, source := socket(family, local)
		fmt.Println("family", family == syscall.AF_INET)
		packet := []byte{0, 128, 255, 42}
		if err := syscall.Sendto(sender, packet, 0, destination); err != nil {
			panic(err)
		}
		storage := []byte{7, 7, 7, 7, 7, 7, 7, 7}
		n, from, err := syscall.Recvfrom(receiver, storage[2:6], syscall.MSG_PEEK)
		result("peek", n, from, err)
		fmt.Println("actual storage", storage, "actual sender", same(from, source))
		n, from, err = syscall.Recvfrom(receiver, storage[3:5], syscall.MSG_TRUNC)
		result("truncated", n, from, err)
		fmt.Println("actual truncated storage", storage, "actual sender", same(from, source))
		if err := syscall.Sendto(sender, nil, 0, destination); err != nil {
			panic(err)
		}
		n, from, err = syscall.Recvfrom(receiver, storage[:1], syscall.MSG_PEEK)
		result("empty ready", n, from, err)
		n, from, err = syscall.Recvfrom(receiver, nil, 0)
		result("empty", n, from, err)
		fmt.Println("empty sender", same(from, source))
		if err := syscall.SetNonblock(receiver, true); err != nil {
			panic(err)
		}
		n, from, err = syscall.Recvfrom(receiver, storage, 0)
		result("empty queue", n, from, err)
		fmt.Println("unchanged storage", storage)
		typ, err := syscall.GetsockoptInt(receiver, syscall.SOL_SOCKET, syscall.SO_TYPE)
		if err != nil {
			panic(err)
		}
		fmt.Println("owned", typ == syscall.SOCK_DGRAM)
		if err := syscall.Close(sender); err != nil {
			panic(err)
		}
		if err := syscall.Close(receiver); err != nil {
			panic(err)
		}
	}
	pair, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_DGRAM, 0)
	if err != nil {
		panic(err)
	}
	if err := syscall.Sendto(pair[0], []byte{255, 0, 128}, 0, nil); err != nil {
		panic(err)
	}
	data := make([]byte, 8)
	n, from, err := syscall.Recvfrom(pair[1], data, 0)
	result("unnamed pair", n, from, err)
	fmt.Println("pair bytes", data[:n])
	for _, fd := range pair {
		if err := syscall.Close(fd); err != nil {
			panic(err)
		}
	}
	dir, err := os.MkdirTemp("", "raw-recvfrom")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	if err := os.Chdir(dir); err != nil {
		panic(err)
	}
	receiver, destination := socket(syscall.AF_UNIX, &syscall.SockaddrUnix{Name: "server"})
	sender, source := socket(syscall.AF_UNIX, &syscall.SockaddrUnix{Name: "client"})
	if err := syscall.Sendto(sender, []byte{128, 0}, 0, destination); err != nil {
		panic(err)
	}
	n, from, err = syscall.Recvfrom(receiver, data, 0)
	result("named Unix", n, from, err)
	fmt.Println("actual Unix sender", same(from, source), "Unix bytes", data[:n])
	for _, fd := range []int{sender, receiver} {
		if err := syscall.Close(fd); err != nil {
			panic(err)
		}
	}
	n, from, err = syscall.Recvfrom(-1, nil, 0)
	result("bad fd", n, from, err)
	file, err := os.Create("regular")
	if err != nil {
		panic(err)
	}
	n, from, err = syscall.Recvfrom(int(file.Fd()), data, 0)
	result("regular fd", n, from, err)
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
