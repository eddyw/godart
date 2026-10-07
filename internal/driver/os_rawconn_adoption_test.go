package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestOSRawConnAdoptedPipe(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("Descriptor adoption requires native descriptors")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/rawadoption\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import (
 "errors"
 "fmt"
 "os"
 "syscall"
 "time"
)
func must(err error) { if err != nil { panic(err) } }
func main() {
 p := make([]int,2); must(syscall.Pipe(p))
 must(syscall.SetNonblock(p[0],true)); must(syscall.SetNonblock(p[1],true))
 r,w := os.NewFile(uintptr(p[0]),"adopted-read"),os.NewFile(uintptr(p[1]),"adopted-write")
 defer r.Close(); defer w.Close()
 c,err := r.SyscallConn(); must(err)
 must(c.Control(func(fd uintptr) { var s syscall.Stat_t; must(syscall.Fstat(int(fd),&s)); if s.Mode&syscall.S_IFMT != syscall.S_IFIFO { panic("actual pipe type") } }))
 entered := make(chan struct{}); done := make(chan error)
 go func() { first := true; done <- c.Read(func(fd uintptr) bool {
  b := make([]byte,1); n,err := syscall.Read(int(fd),b)
  if err == syscall.EAGAIN { if first { first = false; close(entered) }; return false }
  must(err); if n != 1 || b[0] != 'a' { panic("adopted read") }; return true
 }) }()
 <-entered; _,err = w.Write([]byte("a")); must(err); must(<-done)
 must(r.SetReadDeadline(time.Now().Add(-time.Second)))
 called := false
 err = c.Read(func(uintptr) bool { called = true; return true })
 if called || !errors.Is(err,os.ErrDeadlineExceeded) { panic("adopted deadline") }
 must(r.SetReadDeadline(time.Time{})); must(r.Close())
 var s syscall.Stat_t
 if syscall.Fstat(p[0],&s) != syscall.EBADF { panic("adopted ownership") }
 fmt.Println("adopted nonblocking pipe readiness and ownership",true)
}
`)
	want := programOutcome(t, dir, "go", "run", ".")
	if want.Exit != 0 {
		t.Fatalf("native Go failed: %+v", want)
	}
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot"} {
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
