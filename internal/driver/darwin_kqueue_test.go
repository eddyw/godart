package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDarwinKqueue(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Kqueue requires macOS")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/kqueue\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

func result(name string, err error) {
	if err == nil {
		fmt.Println(name, "nil")
		return
	}
	code, ok := err.(syscall.Errno)
	fmt.Println(name, int(code), ok)
}
func main() {
	if len(os.Args) > 1 && os.Args[1] == "unavailable" {
		fd, err := syscall.Kqueue()
		code, typed := err.(syscall.Errno)
		match := typed && code == syscall.ENOSYS && errors.Is(err, errors.ErrUnsupported)
		fmt.Println("unavailable", fd, int(code), match)
		if fd != -1 || !match {
			panic("Darwin Kqueue ENOSYS mismatch")
		}
		return
	}

	fd, err := syscall.Kqueue()
	if err != nil {
		panic(err)
	}
	fmt.Println("queue", fd >= 0)
	var stat syscall.Stat_t
	err = syscall.Fstat(fd, &stat)
	result("stat", err)
	if err == nil {
		fmt.Println("type", uint32(stat.Mode)&syscall.S_IFMT)
	}
	result("nonblock", syscall.SetNonblock(fd, true))
	var read syscall.FdSet
	read.Bits[fd/32] |= 1 << uint(fd%32)
	result("select", syscall.Select(fd+1, &read, nil, nil, &syscall.Timeval{}))
	fmt.Println("readable", read.Bits[fd/32]&(1<<uint(fd%32)) != 0)
	duplicate, e := syscall.Dup(fd)
	result("dup", e)
	if e == nil {
		result("close duplicate", syscall.Close(duplicate))
	}
	result("close", syscall.Close(fd))
	result("close again", syscall.Close(fd))
	var original syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &original); err != nil {
		panic(err)
	}
	limit := original
	limit.Cur = 64
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &limit); err != nil {
		panic(err)
	}
	var queues []int
	for {
		next, e := syscall.Kqueue()
		if e != nil {
			result("exhausted", e)
			fmt.Println("failure descriptor", next == -1)
			break
		}
		queues = append(queues, next)
	}
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &original); err != nil {
		panic(err)
	}
	for _, next := range queues {
		if err := syscall.Close(next); err != nil {
			panic(err)
		}
	}
	fmt.Println("exhaustion cleanup", len(queues) > 0)

}
`)
	native := filepath.Join(t.TempDir(), "native")
	run(t, dir, "go", "build", "-o", native, ".")
	want := upstreamOutcome(t, dir, os.Environ(), runLimit, native)
	if want.Exit != 0 || want.Text != "queue true\nstat nil\ntype 4096\nnonblock 25 true\nselect nil\nreadable false\ndup nil\nclose duplicate nil\nclose nil\nclose again 9 true\nexhausted 24 true\nfailure descriptor true\nexhaustion cleanup true" {
		t.Fatalf("native queue contract: %+v", want)
	}
	t.Log(want.Text)
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			var got programResult
			if target == "vm" {
				got = upstreamOutcome(t, dir, os.Environ(), runLimit, "dart", filepath.Join(out, "bin/main.dart"))
			} else {
				got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
			}
			if got != want {
				t.Fatalf("native queue differs: Go=%+v Gart=%+v", want, got)
			}
		})
	}
	for _, target := range []string{"js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			got := runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{"unavailable"})
			if got.Exit != 0 || got.Text != "unavailable -1 78 true" {
				t.Fatalf("web queue differs: %+v", got)
			}
		})
	}
}
