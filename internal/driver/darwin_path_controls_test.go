package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDarwinPathControls(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("path controls require macOS")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/pathcontrols\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func result(name string, err error) {
	if err == nil {
		fmt.Println(name, "nil")
		return
	}
	code, typed := err.(syscall.Errno)
	fmt.Println(name, int(code), typed)
}
func main() {
	if len(os.Args) > 1 && os.Args[1] == "unavailable" {
		valid := true
		for i, call := range []func(string) error{syscall.Revoke, syscall.Undelete} {
			for j, path := range []string{"absent", "", "a\x00b", "\xff"} {
				err := call(path)
				code, typed := err.(syscall.Errno)
				expected := syscall.ENOSYS
				if j == 2 {
					expected = syscall.EINVAL
				}
				match := typed && code == expected
				if expected == syscall.ENOSYS {
					match = match && errors.Is(err, errors.ErrUnsupported)
				}
				fmt.Println("unavailable", i, j, int(code), match)
				valid = valid && match
			}
		}
		if !valid {
			panic("Darwin path control error mismatch")
		}
		return
	}

	dir, err := os.MkdirTemp("", "path-controls-oracle-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "file")
	if err = os.WriteFile(path, []byte{71, 73, 79}, 0600); err != nil {
		panic(err)
	}
	fd, err := syscall.Open(path, syscall.O_RDONLY, 0)
	if err != nil {
		panic(err)
	}
	defer syscall.Close(fd)
	before, err := os.Stat(path)
	if err != nil {
		panic(err)
	}
	for _, p := range []string{path, dir, filepath.Join(dir, "absent"), "", path + "\x00tail", filepath.Join(dir, "absent-\xff")} {
		result("revoke", syscall.Revoke(p))
		result("undelete", syscall.Undelete(p))
	}
	buffer := []byte{11, 13, 17, 19, 23}
	n, err := syscall.Read(fd, buffer[1:4])
	fmt.Println("descriptor", n, err == nil, buffer)
	after, err := os.Stat(path)
	fmt.Println("metadata", err == nil && os.SameFile(before, after) && before.Size() == after.Size() && before.Mode() == after.Mode())
	data, err := os.ReadFile(path)
	fmt.Println("retained", err == nil && string(data) == string([]byte{71, 73, 79}))
}
`)
	native := filepath.Join(t.TempDir(), "native")
	run(t, dir, "go", "build", "-o", native, ".")
	want := upstreamOutcome(t, dir, os.Environ(), runLimit, native)
	if want.Exit != 0 || want.Text != "revoke 45 true\nundelete 1 true\nrevoke 45 true\nundelete 1 true\nrevoke 2 true\nundelete 1 true\nrevoke 2 true\nundelete 1 true\nrevoke 22 true\nundelete 22 true\nrevoke 2 true\nundelete 1 true\ndescriptor 3 true [11 71 73 79 23]\nmetadata true\nretained true" {
		t.Fatalf("native path contract: %+v", want)
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
				t.Fatalf("native paths differ: Go=%+v Gart=%+v", want, got)
			}
		})
	}
	for _, target := range []string{"js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			got := runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{"unavailable"})
			if got.Exit != 0 || got.Text != "unavailable 0 0 78 true\nunavailable 0 1 78 true\nunavailable 0 2 22 true\nunavailable 0 3 78 true\nunavailable 1 0 78 true\nunavailable 1 1 78 true\nunavailable 1 2 22 true\nunavailable 1 3 78 true" {
				t.Fatalf("web paths differ: %+v", got)
			}
		})
	}
}
