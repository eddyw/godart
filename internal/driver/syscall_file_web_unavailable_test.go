package driver_test

import (
	"fmt"
	"github.com/eddyw/godart/internal/driver"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSyscallFileWebUnavailable(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("selected syscall SDK requires macOS or Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/fileunavailable\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"syscall"
)

type operation struct {
	name string
	call func() error
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("native file declarations")
		return
	}
	data := []byte{71, 73, 79}
	original := append([]byte(nil), data...)
	stat := syscall.Stat_t{Size: 17}
	originalStat := stat
	calls := []operation{
		{"chdir", func() error { return syscall.Chdir("absent") }},
		{"chmod", func() error { return syscall.Chmod("absent", 0600) }},
		{"chown", func() error { return syscall.Chown("absent", -1, -1) }},
		{"lchown", func() error { return syscall.Lchown("absent", -1, -1) }},
		{"link", func() error { return syscall.Link("absent", "other") }},
		{"symlink", func() error { return syscall.Symlink("absent", "other") }},
		{"rename", func() error { return syscall.Rename("absent", "other") }},
		{"truncate", func() error { return syscall.Truncate("absent", 0) }},
		{"fchdir", func() error { return syscall.Fchdir(-1) }},
		{"fchmod", func() error { return syscall.Fchmod(-1, 0600) }},
		{"fchown", func() error { return syscall.Fchown(-1, -1, -1) }},
		{"ftruncate", func() error { return syscall.Ftruncate(-1, 0) }},
		{"pread", func() error {
			n, e := syscall.Pread(-1, data, 0)
			if n != -1 {
				panic("pread count")
			}
			return e
		}},
		{"pwrite", func() error {
			n, e := syscall.Pwrite(-1, data, 0)
			if n != -1 {
				panic("pwrite count")
			}
			return e
		}},
		{"mkdir", func() error { return syscall.Mkdir("absent", 0700) }},
		{"read", func() error {
			n, e := syscall.Read(-1, data)
			if n != -1 {
				panic("read count")
			}
			return e
		}},
		{"write", func() error {
			n, e := syscall.Write(-1, data)
			if n != -1 {
				panic("write count")
			}
			return e
		}},
		{"stat", func() error { return syscall.Stat("absent", &stat) }},
		{"lstat", func() error { return syscall.Lstat("absent", &stat) }},
		{"fstat", func() error { return syscall.Fstat(-1, &stat) }},
		{"os-pipe", func() error { _, _, e := os.Pipe(); return e }},
		{"os-openroot", func() error { _, e := os.OpenRoot("absent"); return e }},
	}
	valid := true
	for _, value := range calls {
		err := value.call()
		var code syscall.Errno
		match := errors.As(err, &code) && code == syscall.ENOSYS && errors.Is(err, errors.ErrUnsupported)
		fmt.Println(value.name, int(code), match)
		valid = valid && match
	}
	if !bytes.Equal(data, original) || stat != originalStat {
		panic("caller data changed")
	}
	if !valid {
		panic("selected SDK file ENOSYS mismatch")
	}
	fmt.Println("all unavailable files", len(calls))
}
`)
	native := filepath.Join(t.TempDir(), "native")
	run(t, dir, "go", "build", "-o", native, ".")
	want := upstreamOutcome(t, dir, os.Environ(), runLimit, native)
	if want.Exit != 0 || want.Text != "native file declarations" {
		t.Fatalf("native declarations: %+v", want)
	}
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	names := []string{"chdir", "chmod", "chown", "lchown", "link", "symlink", "rename", "truncate", "fchdir", "fchmod", "fchown", "ftruncate", "pread", "pwrite", "mkdir", "read", "write", "stat", "lstat", "fstat", "os-pipe", "os-openroot"}
	code := 38
	if runtime.GOOS == "darwin" {
		code = 78
	}
	var lines []string
	for _, name := range names {
		lines = append(lines, fmt.Sprint(name, " ", code, " true"))
	}
	lines = append(lines, fmt.Sprint("all unavailable files ", len(names)))
	expected := strings.Join(lines, "\n")
	t.Log("native file declarations", len(names))
	for _, target := range []string{"js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			got := runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{"unavailable"})
			if got.Exit != 0 || got.Text != expected {
				t.Fatalf("selected SDK file ENOSYS differs: %+v", got)
			}
		})
	}
}
