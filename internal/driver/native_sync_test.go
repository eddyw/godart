package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestNativeSyncSyscalls(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native sync syscalls require a native filesystem")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/nativesync\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
	"fmt"
	"io"
	"os"
	"syscall"
)

type syncCall struct {
	name string
	call func(int) error
}

var syncCalls = []syncCall{{"fsync", syscall.Fsync}}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func result(label string, err error) {
	code, typed := err.(syscall.Errno)
	fmt.Println(label, int(code), typed)
}
func main() {
	root, err := os.MkdirTemp("", "native-sync")
	must(err)
	defer os.RemoveAll(root)
	for _, operation := range syncCalls {
		fmt.Println(operation.name)
		file, err := os.CreateTemp(root, "file")
		must(err)
		_, err = file.WriteString("contents")
		must(err)
		fd := int(file.Fd())
		result("file", operation.call(fd))
		position, err := file.Seek(0, io.SeekCurrent)
		must(err)
		fmt.Println("position", position)
		var stat syscall.Stat_t
		fmt.Println("owned", syscall.Fstat(fd, &stat) == nil)
		must(file.Close())
		result("closed", operation.call(fd))
		result("invalid", operation.call(-1))
		file, err = os.Open(file.Name())
		must(err)
		result("readonly", operation.call(int(file.Fd())))
		must(file.Close())
		directory, err := os.Open(root)
		must(err)
		result("directory", operation.call(int(directory.Fd())))
		must(directory.Close())
		device, err := os.Open(os.DevNull)
		must(err)
		result("device", operation.call(int(device.Fd())))
		must(device.Close())
	}
}
`)
	writeFile(t, filepath.Join(dir, "sync_linux.go"), `//go:build linux

package main

import "syscall"

func init() { syncCalls = append(syncCalls, syncCall{"fdatasync", syscall.Fdatasync}) }
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
