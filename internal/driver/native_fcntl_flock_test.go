//go:build darwin || linux

package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
)

func TestNativeFcntlFlock(t *testing.T) {
	const fixture = `package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func result(label string, err error) {
	if err == nil {
		fmt.Println(label, 0)
		return
	}
	code, ok := err.(syscall.Errno)
	if !ok {
		panic("non-errno")
	}
	fmt.Println(label, int(code))
}
func main() {
	if len(os.Args) > 1 {
		fd, err := syscall.Open(os.Args[1], syscall.O_RDWR, 0)
		must(err)
		defer syscall.Close(fd)
		value := syscall.Flock_t{Type: syscall.F_WRLCK, Whence: 1, Start: 31414, Len: 271829}
		setPadding(&value)
		err = syscall.FcntlFlock(uintptr(fd), syscall.F_GETLK, &value)
		must(err)
		fmt.Println("actual parent lock", value.Type == syscall.F_WRLCK, value.Whence == 0, value.Start == 31415, value.Len == 271828, value.Pid == int32(syscall.Getppid()))
		return
	}
	dir, err := os.MkdirTemp("", "fcntl-lock-contract-")
	must(err)
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "file")
	fd, err := syscall.Open(path, syscall.O_CREAT|syscall.O_RDWR, 0600)
	must(err)
	_, err = syscall.Write(fd, []byte{0, 128, 255})
	must(err)
	_, err = syscall.Seek(fd, 2, 0)
	must(err)
	for i, args := range [][4]int64{
		{int64(syscall.F_RDLCK), 0, 0, 0}, {int64(syscall.F_WRLCK), 1, 1, 0}, {int64(syscall.F_WRLCK), 2, 0, 1},
		{int64(syscall.F_WRLCK), 0, 1<<33 | 19, 1<<31 | 23}, {int64(syscall.F_WRLCK), 0, 71, -7},
		{int64(syscall.F_WRLCK), 0, -1, 0}, {int64(syscall.F_WRLCK), 0, 1<<63 - 1, 1},
		{int64(syscall.F_WRLCK), -1, 0, 0}, {-1, 0, 0, 0}, {int64(syscall.F_WRLCK), 0, 0, -1},
	} {
		original := syscall.Flock_t{Type: int16(args[0]), Whence: int16(args[1]), Start: args[2], Len: args[3], Pid: 71}
		setPadding(&original)
		values := [3]syscall.Flock_t{original, original, original}
		alias := &values[1]
		copy := values[1]
		result(fmt.Sprint("get ", i), syscall.FcntlFlock(uintptr(fd), syscall.F_GETLK, alias))
		fmt.Println("record", values[1])
		fmt.Println("guards", values[0] == original, values[2] == original, copy == original, alias == &values[1])
	}
	for _, kind := range []int16{syscall.F_RDLCK, syscall.F_WRLCK} {
		value := syscall.Flock_t{Type: kind, Whence: 0, Start: 1<<33 | 19, Len: 17, Pid: 73}
		setPadding(&value)
		old := value
		result("set", syscall.FcntlFlock(uintptr(fd), syscall.F_SETLK, &value))
		fmt.Println("retained", value == old)
		other, err := syscall.Open(path, syscall.O_RDWR, 0)
		must(err)
		query := old
		result("same process", syscall.FcntlFlock(uintptr(other), syscall.F_GETLK, &query))
		fmt.Println("unlocked", query.Type == syscall.F_UNLCK)
		must(syscall.Close(other))
		value.Type = syscall.F_UNLCK
		old = value
		result("unlock", syscall.FcntlFlock(uintptr(fd), syscall.F_SETLKW, &value))
		fmt.Println("retained", value == old)
	}
	value := syscall.Flock_t{Type: syscall.F_WRLCK, Whence: 0, Start: 31, Len: 37, Pid: 41}
	setPadding(&value)
	old := value
	result("invalid command", syscall.FcntlFlock(uintptr(fd), -1, &value))
	fmt.Println("retained", value == old)
	result("invalid fd", syscall.FcntlFlock(^uintptr(0), syscall.F_GETLK, &value))
	fmt.Println("retained", value == old)
	result("nil", syscall.FcntlFlock(uintptr(fd), syscall.F_GETLK, nil))
	result("nil invalid fd", syscall.FcntlFlock(^uintptr(0), syscall.F_GETLK, nil))
	result("descriptor width", syscall.FcntlFlock(uintptr(fd)|(1<<32), syscall.F_GETLK, &value))
	fmt.Println("record", value)
	value = old
	result("command width", syscall.FcntlFlock(uintptr(fd), syscall.F_GETLK|(1<<32), &value))
	fmt.Println("record", value)
	reader, err := syscall.Open(path, syscall.O_RDONLY, 0)
	must(err)
	value = old
	result("read-only write lock", syscall.FcntlFlock(uintptr(reader), syscall.F_SETLK, &value))
	fmt.Println("retained", value == old)
	must(syscall.Close(reader))
	position, err := syscall.Seek(fd, 0, 1)
	must(err)
	bytes, err := os.ReadFile(path)
	must(err)
	fmt.Println("position bytes", position, bytes)
	duplicate, err := syscall.Dup(fd)
	must(err)
	must(syscall.Close(fd))
	value = old
	result("duplicate", syscall.FcntlFlock(uintptr(duplicate), syscall.F_GETLK, &value))
	must(syscall.Close(duplicate))
	value = old
	result("closed", syscall.FcntlFlock(uintptr(duplicate), syscall.F_GETLK, &value))
	fmt.Println("retained", value == old)
}
`
	const linuxPadding = `package main
import "syscall"
func setPadding(value *syscall.Flock_t){value.Pad_cgo_0=[4]byte{17,19,23,29};value.Pad_cgo_1=[4]byte{31,37,41,43}}
`
	const darwinPadding = `package main
import "syscall"
func setPadding(value *syscall.Flock_t){}
`
	padding := darwinPadding
	if runtime.GOOS == "linux" {
		padding = linuxPadding
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/accept\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), fixture)
	writeFile(t, filepath.Join(dir, "padding.go"), padding)
	parent := filepath.Join(t.TempDir(), "parent-lock")
	fd, err := syscall.Open(parent, syscall.O_CREAT|syscall.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer syscall.Close(fd)
	if err := syscall.Ftruncate(fd, 1<<20); err != nil {
		t.Fatal(err)
	}
	lock := syscall.Flock_t{Type: syscall.F_WRLCK, Whence: 1, Start: 31415, Len: 271828}
	if err := syscall.FcntlFlock(uintptr(fd), syscall.F_SETLK, &lock); err != nil {
		t.Fatal(err)
	}
	native := filepath.Join(t.TempDir(), "native")
	run(t, dir, "go", "build", "-o", native, ".")
	want := map[string]programResult{}
	for _, mode := range []string{"normal", "parent"} {
		args := []string{}
		if mode == "parent" {
			args = []string{parent}
		}
		want[mode] = upstreamOutcome(t, dir, nil, runLimit, native, args...)
		if want[mode].Exit != 0 {
			t.Fatalf("native FcntlFlock %s: %+v", mode, want[mode])
		}
		if mode == "parent" && want[mode].Text != "actual parent lock true true true true true" {
			t.Fatalf("native parent lock: %+v", want[mode])
		}
		t.Logf("original SDK FcntlFlock %s %s", mode, want[mode].Text)
	}
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			for _, mode := range []string{"normal", "parent"} {
				t.Run(mode, func(t *testing.T) {
					args := []string{}
					if mode == "parent" {
						args = []string{parent}
					}
					var got programResult
					if target == "vm" {
						got = upstreamOutcome(t, dir, nil, runLimit, "dart", append([]string{filepath.Join(out, "bin/main.dart")}, args...)...)
					} else {
						got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, args)
					}
					if got != want[mode] {
						t.Fatalf("FcntlFlock %s/%s: got %+v, want %+v", target, mode, got, want[mode])
					}
				})
			}
		})
	}
	web, webout := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(web, "go.mod"), "module example.com/accept\n\ngo 1.27\n")
	writeFile(t, filepath.Join(web, "padding.go"), padding)
	writeFile(t, filepath.Join(web, "main.go"), `package main
import("errors";"fmt";"syscall")
func main(){
 for _,command:=range []int{syscall.F_GETLK,syscall.F_SETLK,syscall.F_SETLKW} {
  value:=syscall.Flock_t{Start:9223372036854775807,Len:-9223372036854775807,Pid:-2147483647,Type:-32767,Whence:32767};setPadding(&value);old:=value
  err:=syscall.FcntlFlock(^uintptr(0),command,&value);var code syscall.Errno
  fmt.Println("input",errors.As(err,&code),code==syscall.ENOSYS,errors.Is(err,errors.ErrUnsupported),value==old)
  err=syscall.FcntlFlock(^uintptr(0),command,nil);fmt.Println("nil",errors.As(err,&code),code==syscall.ENOSYS,errors.Is(err,errors.ErrUnsupported))
 }
}`)
	if err := driver.Build(driver.Options{Dir: web, Patterns: []string{"."}, Out: webout, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, webout)
	for _, target := range []string{"js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			got := runUpstreamTarget(t, target, webout, "bin/main.dart", web, []string{})
			expected := "input true true true true\nnil true true true\ninput true true true true\nnil true true true\ninput true true true true\nnil true true true"
			if got.Exit != 0 || got.Text != expected {
				t.Fatalf("web FcntlFlock %s: got %+v, want %s", target, got, expected)
			}
		})
	}
}
