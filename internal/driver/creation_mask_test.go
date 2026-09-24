package driver_test

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestFileCreationMask(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file creation masks require a Unix host")
	}
	if _, err := exec.LookPath("sh"); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module creationmask\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
 "fmt"
 "os"
 "path/filepath"
)

func main() {
 dir, err := os.MkdirTemp("", "creation-mask-")
 if err != nil { panic(err) }
 defer os.RemoveAll(dir)
 for i, perm := range []os.FileMode{0600, 0644, 0666} {
  path := filepath.Join(dir, fmt.Sprint(i))
  f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, perm)
  if err != nil { panic(err) }
  if err = f.Close(); err != nil { panic(err) }
  info, err := os.Stat(path)
  if err != nil { panic(err) }
  fmt.Printf("open %o\n", info.Mode().Perm())
  path += "-write"
  if err := os.WriteFile(path, []byte("x"), perm); err != nil { panic(err) }
  info, err = os.Stat(path)
  if err != nil { panic(err) }
  fmt.Printf("write %o\n", info.Mode().Perm())
  link := path + "-link"
  target := path + "-target"
  if err := os.Symlink(target, link); err != nil { panic(err) }
  if err := os.WriteFile(link, []byte("target"), perm); err != nil { panic(err) }
  info, err = os.Stat(target)
  if err != nil { panic(err) }
  fmt.Printf("write-link %o\n", info.Mode().Perm())
 }
 f, err := os.CreateTemp(dir, "temp")
 if err != nil { panic(err) }
 if err = f.Close(); err != nil { panic(err) }
 info, err := os.Stat(f.Name())
 if err != nil { panic(err) }
 fmt.Printf("temp %o\n", info.Mode().Perm())
}
`)
	native := filepath.Join(t.TempDir(), "native")
	run(t, dir, "go", "build", "-o", native, ".")
	out := t.TempDir()
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	exe := filepath.Join(out, "program")
	run(t, out, "dart", "compile", "exe", "bin/main.dart", "-o", exe)
	for _, mask := range []string{"000", "002", "022", "077"} {
		t.Run(mask, func(t *testing.T) {
			prefix := []string{"-c", `umask "$1"; shift; exec "$@"`, "creation-mask", mask}
			want := runProgram(t, dir, "sh", append(prefix, native)...)
			for _, target := range []struct {
				name string
				args []string
			}{{"vm", []string{"dart", "bin/main.dart"}}, {"aot", []string{exe}}} {
				t.Run(target.name, func(t *testing.T) {
					got := runProgram(t, out, "sh", append(prefix, target.args...)...)
					if got != want {
						t.Fatalf("file modes differ\nGo:\n%s\nGart:\n%s", want, got)
					}
				})
			}
		})
	}
}
