package driver_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestRawDirectoryNames(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("raw Unix paths require Linux or macOS")
	}
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module rawdirectorynames\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
 "fmt"
 "os"
 "path/filepath"
)

func main() {
 for _, base := range []string{"ascii", "unicode-世界", "raw-\xff", "empty", "absent", "file"} {
  entries, err := os.ReadDir(filepath.Join(os.Args[1], base))
  fmt.Printf("dir %x %t\n", base, err != nil)
  for _, entry := range entries {
   info, err := entry.Info()
   fmt.Printf("entry %x %t %t %t", entry.Name(), entry.IsDir(), entry.Type()&os.ModeSymlink != 0, err != nil)
   if err == nil { fmt.Printf(" %x", info.Name()) }
   fmt.Println()
  }
 }
}
`)
	native := filepath.Join(t.TempDir(), "native")
	run(t, dir, "go", "build", "-o", native, ".")
	files := t.TempDir()
	writeFile(t, filepath.Join(files, "file"), "regular")
	for _, base := range []string{"ascii", "unicode-世界", "raw-\xff", "raw-\ufffd", "empty"} {
		if runtime.GOOS == "darwin" && base == "raw-\xff" {
			continue
		}
		parent := filepath.Join(files, base)
		if err := os.Mkdir(parent, 0700); err != nil {
			t.Fatal(err)
		}
		if base == "empty" {
			continue
		}
		if base == "raw-\ufffd" {
			writeFile(t, filepath.Join(parent, "decoy"), "decoy")
			continue
		}
		for _, name := range []string{"ascii", "unicode-世界", "raw-\xff", "raw-\ufffd"} {
			if runtime.GOOS == "darwin" && name == "raw-\xff" {
				continue
			}
			writeFile(t, filepath.Join(parent, name), "value")
			if err := os.Mkdir(filepath.Join(parent, name+"-dir"), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("absent", filepath.Join(parent, name+"-link")); err != nil {
				t.Fatal(err)
			}
		}
	}
	want := runProgram(t, dir, native, files)
	out := t.TempDir()
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	exe := filepath.Join(out, "program")
	run(t, out, "dart", "compile", "exe", "bin/main.dart", "-o", exe)
	for _, target := range []struct {
		name    string
		command string
		args    []string
	}{{"vm", "dart", []string{"bin/main.dart"}}, {"aot", exe, nil}} {
		t.Run(target.name, func(t *testing.T) {
			if got := runProgram(t, out, target.command, append(target.args, files)...); got != want {
				t.Fatalf("directory names differ\nGo:\n%s\nGart:\n%s", want, got)
			}
		})
	}
}
