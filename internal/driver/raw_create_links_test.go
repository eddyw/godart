package driver_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestRawCreateLinks(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("raw Unix paths require Linux or macOS")
	}
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module rawcreatelinks\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
 "errors"
 "fmt"
 "os"
 "path/filepath"
 "syscall"
)

func main() {
 for _, name := range []string{"ascii", "unicode-世界", "raw-\xff", "missing", "bad\x00"} {
  old := "target-" + name
  var from string
  var err error
  to := filepath.Join(os.Args[1], "link-"+name)
  if os.Args[2] == "symlink" {
   from = old
   err = os.Symlink(from, to)
  } else {
   from = filepath.Join(os.Args[1], old)
   err = os.Link(from, to)
  }
  fmt.Printf("%x %t %t %t %t\n", name, err != nil, errors.Is(err, syscall.ENOENT), errors.Is(err, syscall.EEXIST), errors.Is(err, syscall.EINVAL))
  if err != nil {
   e, ok := err.(*os.LinkError)
   fmt.Println("link error", ok, e.Old == from, e.New == to, e.Op == os.Args[2])
   _, errno := e.Err.(syscall.Errno)
   fmt.Println("errno", errno)
  }
 }
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
	for _, operation := range []string{"symlink", "link"} {
		t.Run(operation, func(t *testing.T) {
			execute := func(command string, args ...string) string {
				t.Helper()
				files := t.TempDir()
				for _, name := range []string{"ascii", "unicode-世界", "raw-\xff", "raw-\ufffd"} {
					err := os.WriteFile(filepath.Join(files, "target-"+name), []byte(name), 0600)
					if err != nil && !(runtime.GOOS == "darwin" && errors.Is(err, syscall.EILSEQ)) {
						t.Fatal(err)
					}
				}
				writeFile(t, filepath.Join(files, "link-raw-\ufffd"), "decoy")
				result := runProgram(t, out, command, append(args, files, operation)...)
				entries, err := os.ReadDir(files)
				if err != nil {
					t.Fatal(err)
				}
				var state strings.Builder
				for _, entry := range entries {
					var data []byte
					if entry.Type()&os.ModeSymlink != 0 {
						target, err := os.Readlink(filepath.Join(files, entry.Name()))
						if err != nil {
							t.Fatal(err)
						}
						data = []byte(target)
					} else {
						data, err = os.ReadFile(filepath.Join(files, entry.Name()))
						if err != nil {
							t.Fatal(err)
						}
					}
					fmt.Fprintf(&state, "%x %t %x\n", entry.Name(), entry.Type()&os.ModeSymlink != 0, data)
				}
				return result + "\n" + state.String()
			}
			want := execute(native)
			for _, target := range []struct {
				name, command string
				args          []string
			}{{"vm", "dart", []string{"bin/main.dart"}}, {"aot", exe, nil}} {
				t.Run(target.name, func(t *testing.T) {
					if got := execute(target.command, target.args...); got != want {
						t.Fatalf("%s differs\nGo:\n%s\nGart:\n%s", operation, want, got)
					}
				})
			}
		})
	}
}
