package driver_test

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestSyscallPositionalIO(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("native directory syscalls require Linux or macOS")
	}
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module syscallposition\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
 "fmt"
 "os"
 "syscall"
)

func result(label string, n int, err error, buf []byte) {
 code, typed := err.(syscall.Errno)
 fmt.Printf("%s %d %d %t %x\n", label, n, code, typed, buf)
}

func probe(fd int) {
 for _, offset := range []int64{-1, 0, 2, 5, 6, 1<<33+7} {
  for _, size := range []int{0, 1, 3, 8} {
   data := make([]byte, size+4)
   for i := range data { data[i] = 0xCD }
   n, err := syscall.Pread(fd, data[2:2+size], offset)
   result("read", n, err, data)
  }
 }
 n, err := syscall.Pread(fd, nil, 0)
 result("nil read", n, err, nil)
}

func main() {
 f, err := os.CreateTemp("", "syscall-position-")
 if err != nil { panic(err) }
 defer os.Remove(f.Name())
 if _, err := f.WriteString("abcdef"); err != nil { panic(err) }
 fd := int(f.Fd())
 probe(fd)
 position, err := f.Seek(0, 1)
 fmt.Println("read offset", position, err == nil)
 for _, offset := range []int64{-1, 0, 2, 6} {
  for _, size := range []int{0, 1, 3} {
   data := []byte("#XYZ#")
   n, err := syscall.Pwrite(fd, data[1:1+size], offset)
   result("write", n, err, data)
  }
 }
 n, err := syscall.Pwrite(fd, nil, 0)
 result("nil write", n, err, nil)
 position, err = f.Seek(0, 1)
 fmt.Println("write offset", position, err == nil)
 data, err := os.ReadFile(f.Name())
 if err != nil { panic(err) }
 fmt.Printf("contents %x\n", data)
 if err := f.Close(); err != nil { panic(err) }
 for _, invalid := range []int{fd, -1} {
  probe(invalid)
  n, err = syscall.Pwrite(invalid, []byte("x"), 0)
  result("invalid write", n, err, nil)
 }
 for _, flags := range []int{syscall.O_RDONLY, syscall.O_WRONLY, syscall.O_RDWR|syscall.O_APPEND} {
  fd, err := syscall.Open(f.Name(), flags, 0)
  if err != nil { panic(err) }
  n, err := syscall.Pwrite(fd, []byte("z"), 0)
  result("flag write", n, err, nil)
  data := make([]byte, 2)
  n, err = syscall.Pread(fd, data, 0)
  result("flag read", n, err, data)
  if err := syscall.Close(fd); err != nil { panic(err) }
 }
 for _, p := range []byte{0, 1} {
  data := make([]byte, int(p))
  n, err := syscall.Pread(0, data, 0)
  result("pipe read", n, err, data)
  n, err = syscall.Pwrite(1, data, 0)
  result("pipe write", n, err, data)
 }
}

`)
	native := filepath.Join(t.TempDir(), "native")
	run(t, dir, "go", "build", "-o", native, ".")
	execute := func(dir, command string, args ...string) string {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), runLimit)
		defer cancel()
		cmd := exec.CommandContext(ctx, command, args...)
		cmd.Dir = dir
		cmd.Stdin = strings.NewReader("")
		data, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s: %v\n%s", command, err, data)
		}
		return string(data)
	}
	want := execute(dir, native)
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
			if got := execute(out, target.command, target.args...); got != want {
				t.Fatalf("syscalls differ\nGo:\n%s\nGart:\n%s", want, got)
			}
		})
	}
}
