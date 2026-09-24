package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestSyscallDirectoryOperations(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("native directory syscalls require Linux or macOS")
	}
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module syscalldirectory\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
 "errors"
 "fmt"
 "os"
 "path/filepath"
 "slices"
 "syscall"
)

func result(label string, n int, err error) {
 _, typed := err.(syscall.Errno)
 fmt.Println(label, n, err != nil, typed, errors.Is(err, syscall.EINVAL), errors.Is(err, syscall.ENOENT), errors.Is(err, syscall.EEXIST), errors.Is(err, syscall.EBADF), errors.Is(err, syscall.ENOTDIR))
}

func main() {
 root, err := os.MkdirTemp("", "syscall-dir-")
 if err != nil { panic(err) }
 defer os.RemoveAll(root)
 for _, name := range []string{"ascii", "raw-\xff", "raw-\ufffd", "bad\x00", "absent/file"} {
  path := filepath.Join(root, name)
  fd, err := syscall.Open(path, syscall.O_CREAT|syscall.O_EXCL|syscall.O_WRONLY, 0600)
  fmt.Printf("open %x %t\n", name, fd >= 0)
  result("open error", 0, err)
  if err != nil { continue }
  var st syscall.Stat_t
  if err := syscall.Fstat(fd, &st); err != nil { panic(err) }
  fmt.Printf("mode %o\n", st.Mode & 0777)
  result("close", 0, syscall.Close(fd))
  result("close again", 0, syscall.Close(fd))
  fd, err = syscall.Open(path, syscall.O_CREAT|syscall.O_EXCL|syscall.O_WRONLY, 0600)
  result("exclusive", fd, err)
 }
 fd, err := syscall.Open(filepath.Join(root, "ascii"), syscall.O_RDONLY, 0)
 if err != nil { panic(err) }
 n, err := syscall.ReadDirent(fd, make([]byte, 256))
 result("regular file", n, err)
 syscall.Close(fd)
 n, err = syscall.ReadDirent(-1, nil)
 result("bad descriptor", n, err)
 fd, err = syscall.Open(root, syscall.O_RDONLY, 0)
 if err != nil { panic(err) }
 for _, size := range []int{0, 1} {
  n, err = syscall.ReadDirent(fd, make([]byte, size))
  result("small buffer", n, err)
 }
 moved := root + "-moved"
 if err := os.Rename(root, moved); err != nil { panic(err) }
 defer os.RemoveAll(moved)
 var names []string
 for {
  data := make([]byte, 272)
  for i := range data { data[i] = 0xCD }
  n, err := syscall.ReadDirent(fd, data[8:264])
  if err != nil { panic(err) }
  for _, b := range data[:8] { if b != 0xCD { panic("prefix overwritten") } }
  for _, b := range data[8+n:] { if b != 0xCD { panic("suffix overwritten") } }
  if n == 0 { break }
  consumed, _, got := syscall.ParseDirent(data[8:8+n], -1, nil)
  if consumed != n { panic("incomplete record") }
  names = append(names, got...)
 }
 slices.Sort(names)
 for _, name := range names { fmt.Printf("entry %x\n", name) }
 n, err = syscall.ReadDirent(fd, make([]byte, 256))
 result("end", n, err)
 result("directory close", 0, syscall.Close(fd))
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
	}{{"vm", "dart", []string{"bin/main.dart"}}, {"aot", exe, nil}} {
		t.Run(target.name, func(t *testing.T) {
			if got := runProgram(t, out, target.command, target.args...); got != want {
				t.Fatalf("syscalls differ\nGo:\n%s\nGart:\n%s", want, got)
			}
		})
	}
}
