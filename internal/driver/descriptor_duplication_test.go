package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestDescriptorDuplication(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("descriptor duplication requires a native filesystem")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/duplication\n\ngo 1.27\n")
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
	base, err := os.MkdirTemp("", "duplicate-contract")
	must(err)
	defer os.RemoveAll(base)
	name := base + "/data"
	must(os.WriteFile(name, []byte("abcdef"), 0600))
	fd, err := syscall.Open(name, syscall.O_RDWR, 0)
	must(err)
	duplicate, err := syscall.Dup(fd)
	must(err)
	fmt.Println("distinct", fd != duplicate)
	first, second := os.NewFile(uintptr(fd), name), os.NewFile(uintptr(duplicate), "duplicate")
	data := make([]byte, 2)
	_, err = first.Read(data)
	must(err)
	fmt.Printf("first %q\n", data)
	_, err = second.Read(data)
	must(err)
	fmt.Printf("shared %q\n", data)
	must(first.Close())
	_, err = second.Seek(0, io.SeekStart)
	must(err)
	_, err = second.Read(data)
	must(err)
	fmt.Printf("survives %q\n", data)
	target, err := syscall.Open(base+"/target", syscall.O_RDWR|syscall.O_CREAT, 0600)
	must(err)
	must(syscall.Dup2(duplicate, target))
	third := os.NewFile(uintptr(target), "target")
	_, err = third.Read(data)
	must(err)
	fmt.Printf("replaced %q\n", data)
	info, err := third.Stat()
	must(err)
	source, err := second.Stat()
	must(err)
	fmt.Println("same", os.SameFile(info, source))
	must(syscall.Dup2(target, target))
	fmt.Println("same descriptor remains", third.Fd() == uintptr(target))
	must(second.Close())
	must(third.Close())
	bad, err := syscall.Dup(-1)
	fmt.Println("invalid source", bad == -1, errors.Is(err, syscall.EBADF))
	fmt.Println("invalid replacement", errors.Is(syscall.Dup2(-1, -1), syscall.EBADF))
	fd, err = syscall.Open(name, syscall.O_RDONLY, 0)
	must(err)
	fmt.Println("invalid source preserves target", errors.Is(syscall.Dup2(-1, fd), syscall.EBADF))
	fmt.Println("invalid target", errors.Is(syscall.Dup2(fd, -1), syscall.EBADF))
	var stat syscall.Stat_t
	fmt.Println("source retained", syscall.Fstat(fd, &stat) == nil)
	must(syscall.Close(fd))
	fd, err = syscall.Open(base, syscall.O_RDONLY, 0)
	must(err)
	duplicate, err = syscall.Dup(fd)
	must(err)
	must(syscall.Close(fd))
	directory := os.NewFile(uintptr(duplicate), base)
	names, err := directory.Readdirnames(-1)
	must(err)
	fmt.Println("directory", len(names) == 2)
	must(directory.Close())
	fd, err = syscall.Open(name, syscall.O_WRONLY|syscall.O_APPEND, 0)
	must(err)
	duplicate, err = syscall.Dup(fd)
	must(err)
	must(syscall.Close(fd))
	appendFile := os.NewFile(uintptr(duplicate), "append-duplicate")
	_, err = appendFile.WriteAt([]byte("x"), 0)
	fmt.Println("append writeat", err)
	_, err = appendFile.WriteString("!")
	must(err)
	must(appendFile.Close())
	contents, err := os.ReadFile(name)
	must(err)
	fmt.Printf("append content %q\n", contents)

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
