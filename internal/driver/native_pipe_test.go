package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestNativePipeBackpressure(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("Pipe validation requires native descriptors")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/pipebackpressure\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
 "bytes"
 "errors"
 "fmt"
 "io"
 "os"
 "syscall"
)

func must(err error) { if err != nil { panic(err) } }
func main() {
 data := make([]byte, 2<<20)
 for i := range data { data[i] = byte(i % 251) }
 r, w, err := os.Pipe()
 must(err)
 for _, f := range []*os.File{r, w} {
  var stat syscall.Stat_t
  must(syscall.Fstat(int(f.Fd()), &stat))
  fmt.Println("native pipe", stat.Mode & syscall.S_IFMT == syscall.S_IFIFO)
 }
 done := make(chan int)
 go func() {
  n, err := w.Write(data)
  must(err)
  must(w.Close())
  done <- n
 }()
 got, err := io.ReadAll(r)
 must(err)
 fmt.Println("transfer", <-done == len(data), bytes.Equal(got, data))
 must(r.Close())
 r, w, err = os.Pipe()
 must(err)
 go func() {
  n, err := w.Write(data)
  if !errors.Is(err, os.ErrClosed) { panic("write did not report close") }
  done <- n
 }()
 prefix := make([]byte, 1)
 _, err = io.ReadFull(r, prefix)
 must(err)
 must(w.Close())
 got, err = io.ReadAll(r)
 must(err)
 got = append(prefix, got...)
 n := <-done
 fmt.Println("partial close", n > 0, n < len(data), n == len(got), bytes.Equal(got, data[:n]))
 must(r.Close())
}
`)
	want := programOutcome(t, dir, "go", "run", ".")
	if want.Exit != 0 {
		t.Fatalf("native Go failed: %+v", want)
	}
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	targets := []string{"vm", "aot"}
	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			var got programResult
			if target == "vm" {
				got = programOutcome(t, dir, "dart", filepath.Join(out, "bin/main.dart"))
			} else {
				got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
			}
			if got != want {
				t.Fatalf("Go:\n%s\nGart (exit %d):\n%s", want.Text, got.Exit, got.Text)
			}
		})
	}
}
