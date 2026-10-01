package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestLinuxDup3(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Dup3 requires Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/linuxdup3\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func main() {
	root, err := os.MkdirTemp("", "dup3-contract")
	must(err)
	defer os.RemoveAll(root)
	source, err := os.CreateTemp(root, "source")
	must(err)
	defer source.Close()
	target, err := os.CreateTemp(root, "target")
	must(err)
	defer target.Close()
	_, err = source.WriteString("abcdef")
	must(err)
	_, err = source.Seek(0, io.SeekStart)
	must(err)
	_, err = target.WriteString("target")
	must(err)
	from, to := int(source.Fd()), int(target.Fd())
	must(syscall.Dup3(from, to, syscall.O_CLOEXEC))
	fmt.Println("cloexec", descriptorFlags(to, syscall.F_GETFD)&syscall.FD_CLOEXEC != 0)
	one, two := make([]byte, 1), make([]byte, 2)
	_, err = source.Read(one)
	must(err)
	_, err = target.Read(two)
	must(err)
	fmt.Printf("shared %q %q\n", one, two)
	must(syscall.Dup3(from, to, 0))
	fmt.Println("cleared", descriptorFlags(to, syscall.F_GETFD)&syscall.FD_CLOEXEC == 0)
	fmt.Println("source flags", descriptorFlags(from, syscall.F_GETFD)&syscall.FD_CLOEXEC != 0)
	original, err := source.Stat()
	must(err)
	for _, c := range []struct {
		name            string
		from, to, flags int
	}{
		{"same", from, from, 0}, {"invalid source", -1, to, 0}, {"invalid target", from, -1, 0},
		{"invalid flags", from, to, syscall.O_APPEND}, {"both invalid", -1, -1, 0},
	} {
		err := syscall.Dup3(c.from, c.to, c.flags)
		fmt.Println(c.name, errors.Is(err, syscall.EINVAL), errors.Is(err, syscall.EBADF))
	}
	info, err := target.Stat()
	must(err)
	fmt.Println("target preserved", os.SameFile(original, info), descriptorFlags(to, syscall.F_GETFD)&syscall.FD_CLOEXEC == 0)
	_, err = source.Seek(0, io.SeekStart)
	must(err)
	must(source.Close())
	contents, err := io.ReadAll(target)
	must(err)
	fmt.Printf("independent close %q\n", contents)
	must(target.Close())
	contents, err = os.ReadFile(target.Name())
	must(err)
	fmt.Printf("target path %q\n", contents)
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
