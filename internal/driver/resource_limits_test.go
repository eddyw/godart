package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestResourceLimits(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("resource limits require a native filesystem")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/resourcelimits\n\ngo 1.27\n")
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
func result(label string, err error) {
	code, typed := err.(syscall.Errno)
	fmt.Println(label, int(code), typed)
}
func main() {
	var original syscall.Rlimit
	must(syscall.Getrlimit(syscall.RLIMIT_NOFILE, &original))
	if original.Cur < 64 {
		panic("resource limit fixture needs at least 64 descriptors")
	}
	fmt.Println("initial", original.Cur <= original.Max)
	result("get nil", syscall.Getrlimit(syscall.RLIMIT_NOFILE, nil))
	result("set nil", syscall.Setrlimit(syscall.RLIMIT_NOFILE, nil))
	value := syscall.Rlimit{Cur: 123, Max: 456}
	result("get invalid", syscall.Getrlimit(-1, &value))
	fmt.Println("output preserved", value.Cur == 123 && value.Max == 456)
	result("get invalid nil", syscall.Getrlimit(-1, nil))
	result("set invalid nil", syscall.Setrlimit(-1, nil))
	invalid := syscall.Rlimit{Cur: 2, Max: 1}
	result("invalid limits", syscall.Setrlimit(syscall.RLIMIT_NOFILE, &invalid))
	var current syscall.Rlimit
	must(syscall.Getrlimit(syscall.RLIMIT_NOFILE, &current))
	fmt.Println("invalid update preserved", current == original)
	reduced := original
	reduced.Cur = 64
	must(syscall.Setrlimit(syscall.RLIMIT_NOFILE, &reduced))
	defer syscall.Setrlimit(syscall.RLIMIT_NOFILE, &original)
	must(syscall.Getrlimit(syscall.RLIMIT_NOFILE, &current))
	fmt.Println("reduced", current == reduced)
	descriptors := []int{}
	var exhausted error
	for range 128 {
		fd, err := syscall.Open(os.DevNull, syscall.O_RDONLY, 0)
		if err != nil {
			exhausted = err
			break
		}
		descriptors = append(descriptors, fd)
	}
	for _, fd := range descriptors {
		must(syscall.Close(fd))
	}
	fmt.Println("kernel limit", errors.Is(exhausted, syscall.EMFILE), len(descriptors) > 0)
	must(syscall.Setrlimit(syscall.RLIMIT_NOFILE, &original))
	must(syscall.Getrlimit(syscall.RLIMIT_NOFILE, &current))
	fmt.Println("restored", current == original)
	var fileSize syscall.Rlimit
	must(syscall.Getrlimit(syscall.RLIMIT_FSIZE, &fileSize))
	must(syscall.Setrlimit(syscall.RLIMIT_FSIZE, &fileSize))
	var same syscall.Rlimit
	must(syscall.Getrlimit(syscall.RLIMIT_FSIZE, &same))
	infinity := int64(syscall.RLIM_INFINITY)
	fmt.Println("file size", fileSize.Cur <= fileSize.Max, fileSize.Max == uint64(infinity), same == fileSize)
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
