package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNativeSocketMessageCalls(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("Native socket message calls require Linux or macOS")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/socketmessage\n\ngo 1.27\n")
	fixture := `package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

const controlFlags = 0

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

func cloexec(fd int) bool {
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
			return value&syscall.O_CLOEXEC != 0
		}
	}
	panic("descriptor flags absent")
}

func receiveRights(fd int, data, oob []byte, originals []int) {
	n, oobn, flags, from, err := syscall.Recvmsg(fd, data, oob, controlFlags)
	if err != nil {
		panic(err)
	}
	fmt.Println("rights receive", n, oobn, flags, from == nil, "data", data)
	messages, err := syscall.ParseSocketControlMessage(oob[:oobn])
	if err != nil {
		panic(err)
	}
	if len(messages) != 1 {
		panic("control message count")
	}
	rights, err := syscall.ParseUnixRights(&messages[0])
	if err != nil {
		panic(err)
	}
	if len(rights) != len(originals) {
		panic("descriptor count")
	}
	fmt.Println("actual header", messages[0].Header.Len, messages[0].Header.Level, messages[0].Header.Type)
	for i, received := range rights {
		fmt.Println("actual descriptor", received >= 0, received != originals[i])
		buffer := make([]byte, 8)
		count, err := syscall.Pread(received, buffer, 0)
		if err != nil {
			panic(err)
		}
		fmt.Println("actual file", count, buffer[:count])
		if runtime.GOOS == "linux" {
			fmt.Println("actual close-on-exec", cloexec(received))
		}
		if err := syscall.Close(received); err != nil {
			panic(err)
		}
	}
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
		count, err := syscall.SendmsgN(sender, []byte{0, 128, 255, 42}, nil, destination, 0)
		fmt.Println("sent", count)
		result("send", err)
		data := []byte{7, 7, 7, 7, 7, 7, 7, 7}
		controls := []byte{9, 9, 9, 9, 9, 9, 9, 9}
		n, oobn, flags, from, err := syscall.Recvmsg(receiver, data[2:4], controls[3:7], syscall.MSG_PEEK|syscall.MSG_TRUNC)
		fmt.Println("peek", n, oobn, flags, same(from, source), data, controls)
		result("peek error", err)
		n, oobn, flags, from, err = syscall.Recvmsg(receiver, data[2:6], controls[3:7], 0)
		fmt.Println("packet", n, oobn, flags, same(from, source), data, controls)
		result("receive", err)
		result("empty send", syscall.Sendmsg(sender, nil, nil, destination, 0))
		n, oobn, flags, from, err = syscall.Recvmsg(receiver, data[:1], nil, syscall.MSG_PEEK)
		fmt.Println("empty ready", n, oobn, flags, same(from, source))
		result("empty ready error", err)
		n, oobn, flags, from, err = syscall.Recvmsg(receiver, nil, nil, 0)
		fmt.Println("empty", n, oobn, flags, same(from, source))
		result("empty error", err)
		if err := syscall.Close(sender); err != nil {
			panic(err)
		}
		if err := syscall.Close(receiver); err != nil {
			panic(err)
		}
	}
	dir, err := os.MkdirTemp("", "raw-message")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	var files []*os.File
	var originals []int
	for i := 0; i < 2; i++ {
		file, err := os.CreateTemp(dir, "source")
		if err != nil {
			panic(err)
		}
		if _, err := file.Write([]byte{byte(i), 128, 255, 0}); err != nil {
			panic(err)
		}
		files = append(files, file)
		originals = append(originals, int(file.Fd()))
	}
	rights := syscall.UnixRights(originals...)
	for _, typ := range []int{syscall.SOCK_DGRAM, syscall.SOCK_STREAM} {
		pair, err := syscall.Socketpair(syscall.AF_UNIX, typ, 0)
		if err != nil {
			panic(err)
		}
		fmt.Println("Unix type", typ == syscall.SOCK_DGRAM)
		count, err := syscall.SendmsgN(pair[0], []byte{0, 128, 255}, rights, nil, 0)
		fmt.Println("rights sent", count)
		result("rights send", err)
		storage := make([]byte, syscall.CmsgSpace(8)+10)
		for i := range storage {
			storage[i] = 0xcc
		}
		data := []byte{7, 7, 7, 7, 7, 7, 7, 7}
		receiveRights(pair[1], data[2:7], storage[5:len(storage)-5], originals)
		fmt.Println("actual storage prefix", storage[:5], "suffix", storage[len(storage)-5:], "data", data)
		count, err = syscall.SendmsgN(pair[0], nil, rights, nil, 0)
		fmt.Println("control-only sent", count)
		result("control-only send", err)
		receiveRights(pair[1], nil, storage[5:len(storage)-5], originals)
		if err := syscall.SetNonblock(pair[1], true); err != nil {
			panic(err)
		}
		n, oobn, flags, from, err := syscall.Recvmsg(pair[1], data, nil, 0)
		fmt.Println("empty queue", n, oobn, flags, from == nil)
		result("empty queue error", err)
		for _, fd := range pair {
			if err := syscall.Close(fd); err != nil {
				panic(err)
			}
		}
	}
	for _, file := range files {
		if err := file.Close(); err != nil {
			panic(err)
		}
	}
	count, err := syscall.SendmsgN(-1, nil, nil, nil, 0)
	fmt.Println("invalid send count", count)
	result("invalid send", err)
	count, err = syscall.SendmsgN(-1, nil, nil, &syscall.SockaddrInet4{Port: -1}, 0)
	fmt.Println("invalid address count", count)
	result("invalid address", err)
	n, oobn, flags, from, err := syscall.Recvmsg(-1, nil, nil, 0)
	fmt.Println("invalid receive", n, oobn, flags, from == nil)
	result("invalid receive error", err)
}
`
	if runtime.GOOS == "linux" {
		fixture = strings.Replace(fixture, "const controlFlags = 0", "const controlFlags = syscall.MSG_CMSG_CLOEXEC", 1)
	}
	writeFile(t, filepath.Join(dir, "main.go"), fixture)
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
