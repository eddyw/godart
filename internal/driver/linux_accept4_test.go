package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLinuxSocketAccept4(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Accept4 requires Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/accept\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
)

func result(label string, err error) {
	code, typed := err.(syscall.Errno)
	fmt.Println(label, code, typed)
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

func descriptorFlags(fd int) uint64 {
	data, err := os.ReadFile(fmt.Sprintf("/proc/self/fdinfo/%d", fd))
	if err != nil {
		panic(err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "flags:" {
			value, err := strconv.ParseUint(fields[1], 8, 64)
			if err != nil {
				panic(err)
			}
			return value
		}
	}
	panic("descriptor flags absent")
}

func main() {
	dir, err := os.MkdirTemp("", "raw-accept")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	if err := os.Chdir(dir); err != nil {
		panic(err)
	}
	for _, flags := range []int{0, syscall.SOCK_NONBLOCK, syscall.SOCK_CLOEXEC, syscall.SOCK_NONBLOCK | syscall.SOCK_CLOEXEC} {
		for _, family := range []int{syscall.AF_INET, syscall.AF_INET6, syscall.AF_UNIX} {
			var local syscall.Sockaddr
			if family == syscall.AF_INET {
				local = &syscall.SockaddrInet4{Addr: [4]byte{127, 0, 0, 1}}
			} else if family == syscall.AF_INET6 {
				local = &syscall.SockaddrInet6{Addr: [16]byte{15: 1}}
			} else {
				local = &syscall.SockaddrUnix{Name: fmt.Sprintf("server-%d", flags)}
			}
			server, err := syscall.Socket(family, syscall.SOCK_STREAM, 0)
			if err != nil {
				panic(err)
			}
			if err := syscall.Bind(server, local); err != nil {
				panic(err)
			}
			destination, err := syscall.Getsockname(server)
			if err != nil {
				panic(err)
			}
			if err := syscall.Listen(server, 4); err != nil {
				panic(err)
			}
			client, err := syscall.Socket(family, syscall.SOCK_STREAM, 0)
			if err != nil {
				panic(err)
			}
			if family == syscall.AF_UNIX {
				if err := syscall.Bind(client, &syscall.SockaddrUnix{Name: fmt.Sprintf("client-%d", flags)}); err != nil {
					panic(err)
				}
			}
			if err := syscall.Connect(client, destination); err != nil {
				panic(err)
			}
			source, err := syscall.Getsockname(client)
			if err != nil {
				panic(err)
			}
			accepted, peer, err := syscall.Accept4(server, flags)
			if err != nil {
				panic(err)
			}
			fmt.Println("flags", flags)
			fmt.Println("family", family == syscall.AF_INET, family == syscall.AF_INET6)
			observed := descriptorFlags(accepted)
			fmt.Println("actual flags", observed&syscall.SOCK_NONBLOCK != 0, observed&syscall.SOCK_CLOEXEC != 0)
			if err := syscall.SetNonblock(accepted, false); err != nil {
				panic(err)
			}
			fmt.Println("actual descriptor", accepted >= 0, "actual peer", same(peer, source))
			got, err := syscall.Getsockname(accepted)
			if err != nil {
				panic(err)
			}
			fmt.Println("actual local", same(got, destination))
			n, err := syscall.Write(client, []byte{0, 128, 255})
			if err != nil {
				panic(err)
			}
			fmt.Println("client write", n)
			data := make([]byte, 8)
			n, err = syscall.Read(accepted, data)
			if err != nil {
				panic(err)
			}
			fmt.Println("accepted read", n, data[:n])
			duplicate, err := syscall.Dup(accepted)
			if err != nil {
				panic(err)
			}
			fmt.Println("actual duplicate flags", descriptorFlags(duplicate)&syscall.SOCK_CLOEXEC != 0)
			if err := syscall.Close(accepted); err != nil {
				panic(err)
			}
			n, err = syscall.Write(duplicate, []byte{255, 0})
			if err != nil {
				panic(err)
			}
			fmt.Println("duplicate write", n)
			n, err = syscall.Read(client, data)
			if err != nil {
				panic(err)
			}
			fmt.Println("client read", n, data[:n])
			if err := syscall.Close(client); err != nil {
				panic(err)
			}
			n, err = syscall.Read(duplicate, data)
			fmt.Println("actual EOF", n)
			result("EOF", err)
			if err := syscall.Close(duplicate); err != nil {
				panic(err)
			}
			if err := syscall.SetNonblock(server, true); err != nil {
				panic(err)
			}
			fd, address, err := syscall.Accept4(server, flags)
			fmt.Println("empty queue", fd, address == nil)
			result("empty queue error", err)
			typ, err := syscall.GetsockoptInt(server, syscall.SOL_SOCKET, syscall.SO_TYPE)
			if err != nil {
				panic(err)
			}
			fmt.Println("listener owned", typ == syscall.SOCK_STREAM)
			if err := syscall.Close(server); err != nil {
				panic(err)
			}
		}
	}
	fd, address, err := syscall.Accept4(-1, 0)
	fmt.Println("bad descriptor", fd, address == nil)
	result("bad descriptor error", err)
	file, err := os.Create("regular")
	if err != nil {
		panic(err)
	}
	fd, address, err = syscall.Accept4(int(file.Fd()), 0)
	fmt.Println("regular descriptor", fd, address == nil)
	result("regular descriptor error", err)
	for _, flags := range []int{1, -1} {
		fd, address, err := syscall.Accept4(-1, flags)
		fmt.Println("invalid flags", fd, address == nil)
		result("invalid flag error", err)
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
