package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestSyscallDescriptorAttributes(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("native directory syscalls require Linux or macOS")
	}
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module syscalldescriptor\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
 "fmt"
 "os"
 "path/filepath"
 "syscall"
)

func result(label string, err error) {
 code, typed := err.(syscall.Errno)
 fmt.Println(label, code, typed)
}

func main() {
 old, err := os.Open(".")
 if err != nil { panic(err) }
 root, err := os.MkdirTemp("", "syscall-fd-")
 if err != nil { panic(err) }
 defer func() { old.Chdir(); old.Close(); os.RemoveAll(root) }()
 fd, err := syscall.Open(root, syscall.O_RDONLY, 0)
 if err != nil { panic(err) }
 result("directory", syscall.Fchdir(fd))
 wd, err := syscall.Getwd()
 if err != nil { panic(err) }
 fmt.Println("directory matches", filepath.Base(wd) == filepath.Base(root))
 result("directory chmod", syscall.Fchmod(fd, 0750))
 var stat syscall.Stat_t
 if err := syscall.Fstat(fd, &stat); err != nil { panic(err) }
 fmt.Printf("directory mode %o\n", stat.Mode & 0777)
 result("directory chown", syscall.Fchown(fd, -1, -1))
 result("directory truncate", syscall.Ftruncate(fd, 0))
 result("directory close", syscall.Close(fd))
 result("closed directory", syscall.Fchdir(fd))
 fd, err = syscall.Open("file", syscall.O_CREAT|syscall.O_RDWR, 0600)
 if err != nil { panic(err) }
 result("file chdir", syscall.Fchdir(fd))
 result("file chmod", syscall.Fchmod(fd, 0640))
 result("file chown unchanged", syscall.Fchown(fd, -1, -1))
 result("file chown self", syscall.Fchown(fd, os.Getuid(), os.Getgid()))
 result("truncate", syscall.Ftruncate(fd, 5))
 if err := syscall.Fstat(fd, &stat); err != nil { panic(err) }
 fmt.Printf("file state %o %d\n", stat.Mode & 0777, stat.Size)
 result("negative length", syscall.Ftruncate(fd, -1))
 result("truncate after rename", syscall.Rename("file", "moved"))
 result("truncate renamed", syscall.Ftruncate(fd, 3))
 data, err := os.ReadFile("moved")
 if err != nil { panic(err) }
 fmt.Printf("renamed contents %x\n", data)
 result("close", syscall.Close(fd))
 for _, invalid := range []int{fd, -1} {
  result("invalid chdir", syscall.Fchdir(invalid))
  result("invalid chmod", syscall.Fchmod(invalid, 0600))
  result("invalid chown", syscall.Fchown(invalid, -1, -1))
  result("invalid truncate", syscall.Ftruncate(invalid, 0))
 }
 fd, err = syscall.Open("moved", syscall.O_RDONLY, 0)
 if err != nil { panic(err) }
 result("readonly truncate", syscall.Ftruncate(fd, 0))
 result("readonly close", syscall.Close(fd))
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
