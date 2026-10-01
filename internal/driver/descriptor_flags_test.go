package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestDescriptorFlags(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("descriptor flags require a native filesystem")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/descriptorflags\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func main() {
	root, err := os.MkdirTemp("", "descriptor-flags")
	must(err)
	defer os.RemoveAll(root)
	fd, err := syscall.Open(root+"/file", syscall.O_RDWR|syscall.O_CREAT|syscall.O_APPEND, 0600)
	must(err)
	defer syscall.Close(fd)
	duplicate, err := syscall.Dup(fd)
	must(err)
	defer syscall.Close(duplicate)
	before := descriptorFlags(fd, syscall.F_GETFL)
	for _, enabled := range []bool{true, true, false, false} {
		must(syscall.SetNonblock(fd, enabled))
		flags := descriptorFlags(fd, syscall.F_GETFL)
		fmt.Println("nonblock", enabled, flags&syscall.O_NONBLOCK != 0, flags&^syscall.O_NONBLOCK == before&^syscall.O_NONBLOCK)
		fmt.Println("shared", descriptorFlags(duplicate, syscall.F_GETFL) == flags)
	}
	fmt.Println("initial cloexec", descriptorFlags(fd, syscall.F_GETFD)&syscall.FD_CLOEXEC != 0)
	syscall.CloseOnExec(fd)
	fmt.Println("cloexec", descriptorFlags(fd, syscall.F_GETFD)&syscall.FD_CLOEXEC != 0)
	fmt.Println("duplicate cloexec", descriptorFlags(duplicate, syscall.F_GETFD)&syscall.FD_CLOEXEC != 0)
	syscall.CloseOnExec(fd)
	fmt.Println("repeat cloexec", descriptorFlags(fd, syscall.F_GETFD)&syscall.FD_CLOEXEC != 0)
	syscall.CloseOnExec(-1)
	fmt.Println("invalid nonblock", errors.Is(syscall.SetNonblock(-1, true), syscall.EBADF))
}
`)
	writeFile(t, filepath.Join(dir, "flags_native.go"), `//go:build !dart

package main

import "syscall"

func descriptorFlags(fd, cmd int) int {
	flags, _, code := syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), uintptr(cmd), 0)
	if code != 0 {
		panic(code)
	}
	return int(flags)
}
`)
	writeFile(t, filepath.Join(dir, "flags_dart.go"), `//go:build dart

package main
import "syscall/dart"
func descriptorFlags(fd, cmd int) int {
 flags, code := dart.SyscallFcntl(fd, cmd, 0)
 if code != 0 { panic(code) }
 return flags
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
	targets := []string{"vm", "aot"}
	for _, target := range targets {
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
