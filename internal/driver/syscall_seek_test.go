package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestSyscallSeek(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("native file syscalls require Linux or macOS")
	}
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module syscallseek\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
 "errors"
 "fmt"
 "os"
 "syscall"
)

func main() {
 f, err := os.CreateTemp("", "syscall-seek-")
 if err != nil { panic(err) }
 defer os.Remove(f.Name())
 if _, err := f.WriteString("abcdef"); err != nil { panic(err) }
 fd := int(f.Fd())
 for _, c := range []struct{offset int64; whence int}{
  {2,0}, {-1,1}, {-2,2}, {1<<54+7,0}, {0,1}, {-1,0}, {0,9}, {0,0},
 } {
  position, err := syscall.Seek(fd,c.offset,c.whence)
  _, typed := err.(syscall.Errno)
  fmt.Println(position, err != nil, typed, errors.Is(err,syscall.EINVAL))
 }
 data := make([]byte,3)
 n, err := f.Read(data)
 fmt.Printf("read %q %d %v\n",data,n,err)
 if err := f.Close(); err != nil { panic(err) }
 for _, fd := range []int{fd,-1} {
  position, err := syscall.Seek(fd,0,0)
  _, typed := err.(syscall.Errno)
  fmt.Println("closed",position,typed,errors.Is(err,syscall.EBADF))
 }
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
				t.Fatalf("Seek differs\nGo:\n%s\nGart:\n%s", want, got)
			}
		})
	}
}
