package driver_test

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestRawWorkingDirectory(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("raw Unix paths require Linux or macOS")
	}
	files := t.TempDir()
	for _, name := range []string{"ascii", "unicode-世界", "raw-\xff", "raw-\ufffd"} {
		path := filepath.Join(files, name)
		if err := os.Mkdir(path, 0700); err != nil {
			if runtime.GOOS == "darwin" && errors.Is(err, syscall.EILSEQ) {
				continue
			}
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(path, "identity"), name)
	}
	writeFile(t, filepath.Join(files, "file"), "data")
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module rawcwd\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
 "errors"
 "fmt"
 "os"
 "path/filepath"
 "syscall"
)

func main() {
 root := os.Args[1]
 for _, name := range []string{"ascii", "unicode-世界", "raw-\xff", "missing", "file", "bad\x00"} {
  if err := os.Chdir(root); err != nil { panic(err) }
  err := os.Chdir(name)
  fmt.Printf("chdir %x %t %t %t %t\n", name, err != nil, errors.Is(err, syscall.ENOENT), errors.Is(err, syscall.ENOTDIR), errors.Is(err, syscall.EINVAL))
  if err != nil {
   e, ok := err.(*os.PathError)
   fmt.Println("path error", ok, e.Op == "chdir", e.Path == name)
   _, typed := e.Err.(syscall.Errno)
   fmt.Println("errno", typed)
   continue
  }
  data, err := os.ReadFile("identity")
  if err != nil { panic(err) }
  fmt.Printf("identity %x\n", data)
  for _, pwd := range []string{"", filepath.Join(root, "raw-\ufffd"), filepath.Join(root, name)} {
   if err := os.Setenv("PWD", pwd); err != nil { panic(err) }
   wd, err := os.Getwd()
   if err != nil { panic(err) }
   fmt.Printf("wd %x\n", filepath.Base(wd))
  }
 }
}
`)
	native := filepath.Join(t.TempDir(), "native")
	run(t, dir, "go", "build", "-o", native, ".")
	want := runProgram(t, dir, native, files)
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
			if got := runProgram(t, out, target.command, append(target.args, files)...); got != want {
				t.Fatalf("working directory differs\nGo:\n%s\nGart:\n%s", want, got)
			}
		})
	}
}
