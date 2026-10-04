package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSyscallMessageValues(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("event value reference requires a selected 64-bit host")
	}
	const fixture = `package main

import (
	"fmt"
	"runtime"
	"syscall"
)

func nilValue(name string, call func()) {
	defer func() {
		value := recover()
		_, typed := value.(runtime.Error)
		fmt.Println("nil", name, value != nil, typed, fmt.Sprint(value))
	}()
	call()
}

func main() {
	data, name := byte(31), byte(37)
	originalIov := syscall.Iovec{Base: &data, Len: 17}
	originalMsg := syscall.Msghdr{Name: &name, Namelen: 41, Iov: &originalIov, Iovlen: 7, Control: &data, Controllen: 19, Flags: -29}
	originalCmsg := syscall.Cmsghdr{Len: 23, Level: -31, Type: 37}
	for i, length := range []int{0, 1, -1, 4294967295, 4294967296, -9223372036854775807, 9223372036854775807} {
		iov, msg, cmsg := originalIov, originalMsg, originalCmsg
		iov.SetLen(length)
		msg.SetControllen(length)
		cmsg.SetLen(length)
		fmt.Println("lengths", i, iov.Len, msg.Controllen, cmsg.Len)
		iov.Len = originalIov.Len
		msg.Controllen = originalMsg.Controllen
		cmsg.Len = originalCmsg.Len
		fmt.Println("retained", i, iov == originalIov, msg == originalMsg, cmsg == originalCmsg, *iov.Base, *msg.Name, *msg.Control, msg.Iov == &originalIov, msg.Iov.Len)
	}
	nilValue("Iovec", func() { var value *syscall.Iovec; value.SetLen(1) })
	nilValue("Msghdr", func() { var value *syscall.Msghdr; value.SetControllen(1) })
	nilValue("Cmsghdr", func() { var value *syscall.Cmsghdr; value.SetLen(1) })
	fmt.Println("original", originalIov.Len, originalMsg.Controllen, originalCmsg.Len, data, name)
}
`
	expected := map[string]string{"darwin": "lengths 0 0 0 0\nretained 0 true true true 31 37 31 true 17\nlengths 1 1 1 1\nretained 1 true true true 31 37 31 true 17\nlengths 2 18446744073709551615 4294967295 4294967295\nretained 2 true true true 31 37 31 true 17\nlengths 3 4294967295 4294967295 4294967295\nretained 3 true true true 31 37 31 true 17\nlengths 4 4294967296 0 0\nretained 4 true true true 31 37 31 true 17\nlengths 5 9223372036854775809 1 1\nretained 5 true true true 31 37 31 true 17\nlengths 6 9223372036854775807 4294967295 4294967295\nretained 6 true true true 31 37 31 true 17\nnil Iovec true true runtime error: invalid memory address or nil pointer dereference\nnil Msghdr true true runtime error: invalid memory address or nil pointer dereference\nnil Cmsghdr true true runtime error: invalid memory address or nil pointer dereference\noriginal 17 19 23 31 37", "linux": "lengths 0 0 0 0\nretained 0 true true true 31 37 31 true 17\nlengths 1 1 1 1\nretained 1 true true true 31 37 31 true 17\nlengths 2 18446744073709551615 18446744073709551615 18446744073709551615\nretained 2 true true true 31 37 31 true 17\nlengths 3 4294967295 4294967295 4294967295\nretained 3 true true true 31 37 31 true 17\nlengths 4 4294967296 4294967296 4294967296\nretained 4 true true true 31 37 31 true 17\nlengths 5 9223372036854775809 9223372036854775809 9223372036854775809\nretained 5 true true true 31 37 31 true 17\nlengths 6 9223372036854775807 9223372036854775807 9223372036854775807\nretained 6 true true true 31 37 31 true 17\nnil Iovec true true runtime error: invalid memory address or nil pointer dereference\nnil Msghdr true true runtime error: invalid memory address or nil pointer dereference\nnil Cmsghdr true true runtime error: invalid memory address or nil pointer dereference\noriginal 17 19 23 31 37"}
	for _, goos := range []string{"darwin", "linux"} {
		for _, arch := range []string{"amd64", "arm64"} {
			t.Run(goos+"/"+arch, func(t *testing.T) {
				reference, dir, out := t.TempDir(), t.TempDir(), t.TempDir()
				if err := os.Mkdir(filepath.Join(reference, "reference"), 0755); err != nil {
					t.Fatal(err)
				}
				writeFile(t, filepath.Join(reference, "go.mod"), "module example.com/eventreference\n\ngo 1.27\n")
				writeFile(t, filepath.Join(reference, "reference", "types.go"), selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/syscall/ztypes_"+goos+"_"+arch+".go"), []string{"Iovec", "Msghdr", "Cmsghdr"}))
				writeFile(t, filepath.Join(reference, "reference", "setter.go"), selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/syscall/syscall_"+goos+"_"+arch+".go"), []string{"Iovec.SetLen", "Msghdr.SetControllen", "Cmsghdr.SetLen"}))
				writeFile(t, filepath.Join(reference, "main.go"), strings.Replace(fixture, "\"syscall\"", "syscall \"example.com/eventreference/reference\"", 1))
				native := filepath.Join(t.TempDir(), "native")
				run(t, reference, "go", "build", "-o", native, ".")
				want := upstreamOutcome(t, reference, os.Environ(), runLimit, native)
				if want.Exit != 0 || want.Text != expected[goos] {
					t.Fatalf("original SDK message values: %+v", want)
				}
				t.Log("original SDK message values", goos, arch, want.Text)
				writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/eventvalues\n\ngo 1.27\n")
				writeFile(t, filepath.Join(dir, "main.go"), fixture)
				t.Setenv("GOOS", goos)
				t.Setenv("GOARCH", arch)
				t.Setenv("CGO_ENABLED", "0")
				if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
					t.Fatal(err)
				}
				prepareDart(t, out)
				for _, target := range []string{"vm", "aot", "js", "wasm"} {
					t.Run(target, func(t *testing.T) {
						var got programResult
						if target == "vm" {
							got = upstreamOutcome(t, dir, os.Environ(), runLimit, "dart", filepath.Join(out, "bin/main.dart"))
						} else {
							got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
						}
						if got != want {
							t.Fatalf("SDK message values differ: Go=%+v Gart=%+v", want, got)
						}
					})
				}
			})
		}
	}
}
