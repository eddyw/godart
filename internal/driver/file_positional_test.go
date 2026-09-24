package driver_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestFilePositionalStdio(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("native positional I/O requires Linux or macOS")
	}
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/positionalstdio\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import (
 "fmt"
 "os"
)
func main() {
 for _, offset := range []int64{-1, 0, 2, 99} {
  data := []byte("XY")
  var n int
  var err error
  if os.Args[1] == "read" {
   n, err = os.Stdin.ReadAt(data, offset)
  } else {
   n, err = os.Stdout.WriteAt(data, offset)
  }
  fmt.Fprintf(os.Stderr, "%d %d %v %q\n", offset, n, err, data)
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
	for _, tc := range []struct{ operation, kind string }{
		{"read", "file"}, {"write", "file"}, {"read", "pipe"}, {"write", "pipe"},
	} {
		t.Run(tc.operation+"/"+tc.kind, func(t *testing.T) {
			execute := func(command string, args ...string) string {
				t.Helper()
				file, err := os.CreateTemp(t.TempDir(), "data")
				if err != nil {
					t.Fatal(err)
				}
				defer file.Close()
				if _, err := file.WriteString("abcdef"); err != nil {
					t.Fatal(err)
				}
				if _, err := file.Seek(3, io.SeekStart); err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithTimeout(context.Background(), runLimit)
				defer cancel()
				cmd := exec.CommandContext(ctx, command, append(args, tc.operation)...)
				cmd.Dir = out
				stream := file
				if tc.kind == "pipe" {
					reader, writer, err := os.Pipe()
					if err != nil {
						t.Fatal(err)
					}
					defer reader.Close()
					defer writer.Close()
					stream = reader
					if tc.operation == "write" {
						stream = writer
					}
				}
				if tc.operation == "read" {
					cmd.Stdin = stream
				} else {
					cmd.Stdout = stream
				}
				var stderr bytes.Buffer
				cmd.Stderr = &stderr
				if err := cmd.Run(); err != nil {
					t.Fatalf("%s: %v\n%s", command, err, stderr.String())
				}
				position, err := file.Seek(0, io.SeekCurrent)
				if err != nil {
					t.Fatal(err)
				}
				data, err := os.ReadFile(file.Name())
				if err != nil {
					t.Fatal(err)
				}
				return fmt.Sprintf("%sposition=%d data=%q", stderr.String(), position, data)
			}
			want := execute(native)
			for _, target := range []struct {
				name    string
				command string
				args    []string
			}{{"vm", "dart", []string{"bin/main.dart"}}, {"aot", exe, nil}} {
				t.Run(target.name, func(t *testing.T) {
					if got := execute(target.command, target.args...); got != want {
						t.Fatalf("positional stdio differs\nGo:\n%s\nGart:\n%s", want, got)
					}
				})
			}
		})
	}
}
