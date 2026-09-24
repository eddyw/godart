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

func TestRawFilePaths(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("raw Unix paths require Linux or macOS")
	}
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module rawfilepaths\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
 "fmt"
 "io"
 "os"
 "path/filepath"
)

func main() {
 for _, name := range []string{"ascii", "unicode-世界", "raw-\xff"} {
  path := filepath.Join(os.Args[1], name)
  switch os.Args[2] {
  case "readfile":
   data, err := os.ReadFile(path)
   fmt.Printf("readfile %x %t %x\n", name, err != nil, data)
  case "writefile":
   err := os.WriteFile(path, []byte("write"), 0600)
   fmt.Printf("writefile %x %t\n", name, err != nil)
  default:
   var f *os.File
   var err error
   switch os.Args[2] {
   case "open": f, err = os.Open(path)
   case "create": f, err = os.Create(path)
   case "openfile": f, err = os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
   }
   fmt.Printf("%s %x %t\n", os.Args[2], name, err != nil)
   if err != nil { continue }
   if os.Args[2] == "open" {
    data, err := io.ReadAll(f)
    fmt.Printf("read %t %x\n", err != nil, data)
   } else {
    n, err := f.Write([]byte("write"))
    fmt.Printf("write %d %t\n", n, err != nil)
   }
   fmt.Println("name", f.Name() == path, "close", f.Close() == nil)
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
	for _, operation := range []string{"readfile", "open", "writefile", "create", "openfile"} {
		t.Run(operation, func(t *testing.T) {
			execute := func(command string, args ...string) string {
				t.Helper()
				files := t.TempDir()
				writeFile(t, filepath.Join(files, "raw-\ufffd"), "decoy")
				if operation == "readfile" || operation == "open" {
					for _, name := range []string{"ascii", "unicode-世界", "raw-\xff"} {
						err := os.WriteFile(filepath.Join(files, name), []byte("seed"), 0600)
						if err != nil && !(runtime.GOOS == "darwin" && name == "raw-\xff" && errors.Is(err, syscall.EILSEQ)) {
							t.Fatal(err)
						}
					}
				}
				result := runProgram(t, out, command, append(args, files, operation)...)
				entries, err := os.ReadDir(files)
				if err != nil {
					t.Fatal(err)
				}
				var state strings.Builder
				for _, entry := range entries {
					info, err := entry.Info()
					if err != nil {
						t.Fatal(err)
					}
					data, err := os.ReadFile(filepath.Join(files, entry.Name()))
					if err != nil {
						t.Fatal(err)
					}
					fmt.Fprintf(&state, "%x %o %x\n", entry.Name(), info.Mode().Perm(), data)
				}
				return result + "\n" + state.String()
			}
			want := execute(native)
			for _, target := range []struct {
				name    string
				command string
				args    []string
			}{{"vm", "dart", []string{"bin/main.dart"}}, {"aot", exe, nil}} {
				t.Run(target.name, func(t *testing.T) {
					if got := execute(target.command, target.args...); got != want {
						t.Fatalf("file paths differ\nGo:\n%s\nGart:\n%s", want, got)
					}
				})
			}
		})
	}
}
