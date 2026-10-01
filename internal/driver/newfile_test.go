package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestNewFileContracts(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("descriptor adoption requires a native filesystem")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/newfilecontracts\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"syscall"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func main() {
	dir, err := os.MkdirTemp("", "newfile-contract")
	must(err)
	defer os.RemoveAll(dir)
	name := dir + "/data"
	must(os.WriteFile(name, []byte("abcdef"), 0600))
	fd, err := syscall.Open(name, syscall.O_RDWR, 0)
	must(err)
	f := os.NewFile(uintptr(fd), "caller-name")
	copy := *f
	fmt.Println("identity", f.Name() == "caller-name", f.Fd() == uintptr(fd))
	data := make([]byte, 2)
	n, err := f.Read(data)
	must(err)
	fmt.Printf("read %d %q\n", n, data)
	pos, err := f.Seek(1, io.SeekStart)
	must(err)
	fmt.Println("seek", pos)
	_, err = f.Write([]byte("Z"))
	must(err)
	_, err = f.WriteAt([]byte("XY"), 3)
	must(err)
	n, err = f.ReadAt(data, 2)
	must(err)
	fmt.Printf("readat %d %q\n", n, data)
	info, err := f.Stat()
	must(err)
	other, err := os.Stat(name)
	must(err)
	fmt.Println("same", os.SameFile(info, other))
	must(f.Sync())
	must(f.Truncate(4))
	must(copy.Close())
	_, err = f.Stat()
	fmt.Println("copy closed", errors.Is(err, os.ErrClosed))
	var stat syscall.Stat_t
	err = syscall.Fstat(fd, &stat)
	fmt.Println("owned", errors.Is(err, syscall.EBADF))
	contents, err := os.ReadFile(name)
	must(err)
	fmt.Printf("contents %q\n", contents)
	fd, err = syscall.Open(name, syscall.O_RDWR|syscall.O_APPEND, 0)
	must(err)
	f = os.NewFile(uintptr(fd), "append")
	_, err = f.WriteAt([]byte("x"), 0)
	fmt.Printf("append writeat %v\n", err)
	_, err = f.WriteString("!")
	must(err)
	must(f.Close())
	contents, err = os.ReadFile(name)
	must(err)
	fmt.Printf("appended %q\n", contents)
	fd, err = syscall.Open(dir, syscall.O_RDONLY, 0)
	must(err)
	f = os.NewFile(uintptr(fd), dir)
	entries, err := f.ReadDir(-1)
	must(err)
	names := []string{}
	for _, entry := range entries {
		info, err := entry.Info()
		must(err)
		names = append(names, fmt.Sprintf("%s:%d", entry.Name(), info.Size()))
	}
	slices.Sort(names)
	fmt.Println("directory", names)
	must(f.Close())
	fmt.Println("negative nil", os.NewFile(^uintptr(0), "negative") == nil)
	f = os.NewFile(1<<30, "invalid-positive")
	fmt.Println("positive nil", f == nil)
	_, err = f.Stat()
	fmt.Println("invalid stat", errors.Is(err, syscall.EBADF))
	n, err = f.Read(data)
	fmt.Println("invalid read", n, errors.Is(err, syscall.EBADF))
	err = f.Close()
	fmt.Println("invalid close", errors.Is(err, syscall.EBADF))
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
