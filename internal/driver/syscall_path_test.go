package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestSyscallPathOperations(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("native directory syscalls require Linux or macOS")
	}
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module syscallpath\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
 "errors"
 "fmt"
 "os"
 "path/filepath"
 "runtime"
 "syscall"
)

func result(label string, err error) {
 code, typed := err.(syscall.Errno)
 fmt.Println(label, code, typed, errors.Is(err, syscall.EINVAL), errors.Is(err, syscall.ENOENT), errors.Is(err, syscall.EEXIST), errors.Is(err, syscall.ENOTDIR))
}

func main() {
 old, err := os.Open(".")
 if err != nil { panic(err) }
 root, err := os.MkdirTemp("", "syscall-path-")
 if err != nil { panic(err) }
 defer func() { old.Chdir(); old.Close(); os.RemoveAll(root) }()
 names := []string{"ascii", "unicode-世界"}
 if runtime.GOOS == "linux" { names = append(names, "raw-\xff") }
 for _, name := range names {
  dir := filepath.Join(root, name)
  if err := os.Mkdir(dir, 0700); err != nil { panic(err) }
  result("chdir", syscall.Chdir(dir))
  wd, err := syscall.Getwd()
  if err != nil { panic(err) }
  fmt.Println("current directory", filepath.Base(wd) == name)
  if err := os.WriteFile("file", []byte("abcdef"), 0600); err != nil { panic(err) }
  result("chmod", syscall.Chmod("file", 0640))
  info, err := os.Stat("file")
  if err != nil { panic(err) }
  fmt.Printf("mode %o\n", info.Mode().Perm())
  result("chown unchanged", syscall.Chown("file", -1, -1))
  result("chown self", syscall.Chown("file", os.Getuid(), os.Getgid()))
  result("link", syscall.Link("file", "hard"))
  result("link exists", syscall.Link("file", "hard"))
  result("symlink", syscall.Symlink("file", "soft"))
  result("symlink exists", syscall.Symlink("file", "soft"))
  result("lchown", syscall.Lchown("soft", -1, -1))
  result("lchown self", syscall.Lchown("soft", os.Getuid(), os.Getgid()))
  result("truncate link", syscall.Truncate("soft", 3))
  data, err := os.ReadFile("hard")
  if err != nil { panic(err) }
  fmt.Printf("contents %q\n", data)
  result("truncate negative", syscall.Truncate("file", -1))
  result("rename", syscall.Rename("file", "moved"))
  result("rename same", syscall.Rename("moved", "moved"))
  result("dangling chown", syscall.Chown("soft", -1, -1))
  result("dangling lchown", syscall.Lchown("soft", -1, -1))
  result("dangling link", syscall.Link("soft", "hard-soft"))
  target, err := os.Readlink("hard-soft")
  fmt.Printf("hard link target %q %t\n", target, err == nil)
  result("rename replace", syscall.Rename("moved", "hard"))
  for _, path := range []string{"", "absent", "hard/child", "bad\x00"} {
   if err := os.WriteFile("hard", []byte("seed"), 0600); err != nil { panic(err) }
   result("chdir invalid", syscall.Chdir(path))
   result("chmod invalid", syscall.Chmod(path, 0600))
   result("chown invalid", syscall.Chown(path, -1, -1))
   result("lchown invalid", syscall.Lchown(path, -1, -1))
   result("truncate invalid", syscall.Truncate(path, 0))
   result("link invalid source", syscall.Link(path, "new"))
   result("link invalid target", syscall.Link("hard", path))
   result("rename invalid source", syscall.Rename(path, "new"))
   result("rename invalid target", syscall.Rename("hard", path))
   result("symlink target", syscall.Symlink("hard", path))
   result("symlink source", syscall.Symlink(path, "new-soft"))
   os.Remove("new")
   os.Remove("new-soft")
   if path == "absent" { os.Remove(path) }
  }
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
	}{{"vm", "dart", []string{"bin/main.dart"}}, {"aot", exe, nil}} {
		t.Run(target.name, func(t *testing.T) {
			if got := runProgram(t, out, target.command, target.args...); got != want {
				t.Fatalf("syscalls differ\nGo:\n%s\nGart:\n%s", want, got)
			}
		})
	}
}
