package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestSyscallGetwd(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("native file syscalls require Linux or macOS")
	}
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module syscallgetwd\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
 "errors"
 "fmt"
 "os"
 "path/filepath"
 "runtime"
 "strings"
 "syscall"
)

func main() {
 old, err := os.Open(".")
 if err != nil { panic(err) }
 root, err := os.MkdirTemp("", "syscall-getwd-")
 if err != nil { panic(err) }
 defer func() { old.Chdir(); old.Close(); os.RemoveAll(root) }()
 if err := os.Chdir(root); err != nil { panic(err) }
 root, err = syscall.Getwd()
 if err != nil { panic(err) }
 names := []string{"ascii", "unicode-世界"}
 if runtime.GOOS == "linux" { names = append(names, "raw-\xff") }
 for _, name := range names {
  path := filepath.Join(root, name)
  if err := os.Mkdir(path, 0700); err != nil { panic(err) }
  if err := os.Chdir(path); err != nil { panic(err) }
  if err := os.Setenv("PWD", "/not-the-current-directory"); err != nil { panic(err) }
  wd, err := syscall.Getwd()
  fmt.Printf("normal %x %t\n", filepath.Base(wd), err == nil)
 }
 if err := os.Chdir(root); err != nil { panic(err) }
 for i := 0; i < 70; i++ {
  name := strings.Repeat("a", 70)
  if err := os.Mkdir(name, 0700); err != nil { panic(err) }
  if err := os.Chdir(name); err != nil { panic(err) }
 }
 wd, err := syscall.Getwd()
 _, typed := err.(syscall.Errno)
 fmt.Println("long", wd == "", typed, errors.Is(err, syscall.ERANGE), errors.Is(err, syscall.EINVAL), errors.Is(err, syscall.ENAMETOOLONG))
 if err := old.Chdir(); err != nil { panic(err) }
 gone := filepath.Join(root, "gone")
 if err := os.Mkdir(gone, 0700); err != nil { panic(err) }
 if err := os.Chdir(gone); err != nil { panic(err) }
 if err := os.Remove(gone); err != nil { panic(err) }
 wd, err = syscall.Getwd()
 _, typed = err.(syscall.Errno)
 fmt.Println("removed", wd == "", typed, errors.Is(err, syscall.ENOENT))
 fmt.Printf("removed-path %q\n", strings.Replace(wd, root, "$ROOT", 1))
}

`)
	native := filepath.Join(t.TempDir(), "native")
	run(t, dir, "go", "build", "-o", native, ".")
	want := runProgram(t, dir, native)
	out := t.TempDir()
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	exe := filepath.Join(out, "program")
	run(t, out, "dart", "compile", "exe", "bin/main.dart", "-o", exe)
	for _, target := range []struct {
		name, command string
		args          []string
	}{
		{"vm", "dart", []string{"bin/main.dart"}}, {"aot", exe, nil},
	} {
		t.Run(target.name, func(t *testing.T) {
			if got := runProgram(t, out, target.command, target.args...); got != want {
				t.Fatalf("Getwd differs\nGo:\n%s\nGart:\n%s", want, got)
			}
		})
	}
}
