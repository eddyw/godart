package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestOSRawConnCallbacks(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("Raw file validation requires native descriptors")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/rawcallbacks\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
 "bytes"
 "errors"
 "fmt"
 "io"
 "os"
 "syscall"
 "time"
)

func must(err error) { if err != nil { panic(err) } }
func assert(ok bool) { if !ok { panic("assertion failed") } }
func pipe() (*os.File, *os.File) { r,w,err := os.Pipe(); must(err); return r,w }
func raw(f *os.File) syscall.RawConn { c,err := f.SyscallConn(); must(err); return c }
func stat(fd uintptr) error { var s syscall.Stat_t; return syscall.Fstat(int(fd), &s) }
func closed(err error) bool { return err != nil && err.Error() == "use of closed file" }

func basic() {
 var f *os.File
 c,err := f.SyscallConn(); assert(c == nil && err == os.ErrInvalid)
 r,w := pipe(); c = raw(r)
 calls := 0
 must(c.Control(func(fd uintptr) { must(stat(fd)); calls++ }))
 must(c.Read(func(fd uintptr) bool { must(stat(fd)); calls++; return true }))
 must(c.Write(func(fd uintptr) bool { must(stat(fd)); calls++; return true }))
 assert(calls == 3)
 func() {
  defer func() { assert(recover() == "callback") }()
  must(c.Control(func(uintptr) { panic("callback") }))
 }()
 for _,action := range []func(){
  func() { must(c.Read(func(uintptr) bool { panic("direction callback") })) },
  func() { must(c.Write(func(uintptr) bool { panic("direction callback") })) },
 } {
  func() { defer func() { assert(recover() == "direction callback") }(); action() }()
 }
 must(c.Read(func(fd uintptr) bool { must(stat(fd)); return true }))
 must(c.Write(func(fd uintptr) bool { must(stat(fd)); return true }))
 must(c.Control(func(fd uintptr) { must(stat(fd)) }))
 must(r.Close()); must(w.Close())
 c,err = r.SyscallConn(); must(err)
 assert(closed(c.Control(func(uintptr) { panic("closed callback") })))
 assert(closed(c.Read(func(uintptr) bool { panic("closed callback") })))
 assert(closed(c.Write(func(uintptr) bool { panic("closed callback") })))
 fmt.Println("basic and panic", true)
}

func readiness() {
 r,w := pipe(); defer r.Close(); defer w.Close()
 c := raw(r); requests := make(chan struct{}); sent := make(chan struct{})
 go func() { for range requests { _,err := w.Write([]byte("q")); must(err); sent <- struct{}{} } }()
 misses,reads := 0,0
 data := make([]byte,1)
 must(c.Read(func(fd uintptr) bool {
  n,err := syscall.Read(int(fd),data)
  if err == syscall.EAGAIN {
   misses++; requests <- struct{}{}; <-sent; return false
  }
  must(err); assert(n == 1 && data[0] == 'q'); reads++
  if reads == 3 { return true }
  n,err = syscall.Read(int(fd),data)
  assert(n == -1 && err == syscall.EAGAIN)
  misses++; requests <- struct{}{}; <-sent; return false
 }))
 close(requests); assert(misses == 3 && reads == 3)
 fmt.Println("actual readiness and repeated false", true)
}

func lifetime() {
 r,w := pipe(); c := raw(r)
 entered,release,done := make(chan uintptr),make(chan struct{}),make(chan error)
 go func() { done <- c.Control(func(fd uintptr) { entered <- fd; <-release; must(stat(fd)) }) }()
 fd := <-entered
 closeDone := make(chan error)
 go func() { closeDone <- r.Close() }()
 time.Sleep(10*time.Millisecond)
 select { case <-closeDone: panic("pollable close returned early"); default: }
 must(stat(fd)); close(release); must(<-done); must(<-closeDone)
 assert(stat(fd) == syscall.EBADF); must(w.Close())

 f,err := os.CreateTemp("", "raw-lifetime"); must(err); defer os.Remove(f.Name())
 c = raw(f); entered,release,done = make(chan uintptr),make(chan struct{}),make(chan error)
 go func() { done <- c.Control(func(fd uintptr) { entered <- fd; <-release; must(stat(fd)) }) }()
 fd = <-entered; must(f.Close()); must(stat(fd)); close(release); must(<-done)
 assert(stat(fd) == syscall.EBADF)
 fmt.Println("suspended callback descriptor lifetime", true)
}

func serialization() {
 r,w := pipe(); defer r.Close(); defer w.Close(); c := raw(r)
 entered,release,done := make(chan struct{}),make(chan struct{}),make(chan error)
 go func() { done <- c.Read(func(uintptr) bool { close(entered); <-release; return true }) }()
 <-entered
 normal := make(chan error)
 go func() { b := make([]byte,1); n,err := r.Read(b); assert(n == 1 && b[0] == 'z'); normal <- err }()
 _,err := w.Write([]byte("z")); must(err)
 must(c.Write(func(fd uintptr) bool { must(stat(fd)); return true }))
 time.Sleep(10*time.Millisecond)
 select { case <-normal: panic("ordinary read bypassed raw lock"); default: }
 close(release); must(<-done); must(<-normal)
 fmt.Println("read serialization and independent write", true)

 entered,release,done = make(chan struct{}),make(chan struct{}),make(chan error)
 cw := raw(w)
 go func() { done <- cw.Write(func(uintptr) bool { close(entered); <-release; return true }) }()
 <-entered
 go func() { _,err := w.Write([]byte("y")); normal <- err }()
 time.Sleep(10*time.Millisecond)
 select { case <-normal: panic("ordinary write bypassed raw lock"); default: }
 close(release); must(<-done); must(<-normal)
 b := make([]byte,1); n,err := r.Read(b); must(err); assert(n == 1 && b[0] == 'y')
 fmt.Println("write serialization", true)
}

func deadlines() {
 r,w := pipe(); defer r.Close(); defer w.Close(); c := raw(r)
 must(r.SetReadDeadline(time.Now().Add(-time.Second)))
 called := false
 err := c.Read(func(uintptr) bool { called = true; return true })
 assert(!called && errors.Is(err,os.ErrDeadlineExceeded))
 must(r.SetReadDeadline(time.Time{}))
 entered := make(chan struct{}); done := make(chan error)
 go func() { first := true; done <- c.Read(func(fd uintptr) bool {
  b := make([]byte,1); n,err := syscall.Read(int(fd),b)
  if err == syscall.EAGAIN { if first { first = false; close(entered) }; return false }
  must(err); assert(n == 1 && b[0] == 'd'); return true
 }) }()
 <-entered; must(r.SetReadDeadline(time.Now().Add(time.Hour))); must(r.SetReadDeadline(time.Time{}))
 _,err = w.Write([]byte("d")); must(err); must(<-done)
 entered = make(chan struct{})
 go func() { done <- c.Read(func(uintptr) bool { close(entered); return false }) }()
 <-entered; must(r.SetReadDeadline(time.Now().Add(-time.Second)))
 assert(errors.Is(<-done,os.ErrDeadlineExceeded))
 must(r.SetReadDeadline(time.Time{}))
 entered = make(chan struct{})
 go func() { done <- c.Read(func(uintptr) bool { close(entered); return false }) }()
 <-entered; must(r.Close()); assert(closed(<-done))
 fmt.Println("deadline changes and close cancellation", true)
}

func regularAndReuse() {
 f,err := os.CreateTemp("", "raw-regular"); must(err); defer os.Remove(f.Name()); c := raw(f)
 calls := 0
 err = c.Read(func(uintptr) bool { calls++; return false })
 assert(calls == 1 && err != nil && err.Error() == "waiting for unsupported file type")
 calls = 0
 err = c.Write(func(uintptr) bool { calls++; return false })
 assert(calls == 1 && err != nil && err.Error() == "waiting for unsupported file type")
 var fd uintptr
 must(c.Control(func(actual uintptr) { fd = actual })); must(f.Close())
 r,w := pipe(); defer r.Close(); defer w.Close()
 var other uintptr
 must(raw(r).Control(func(actual uintptr) { other = actual }))
 if other != fd { must(syscall.Dup2(int(other),int(fd))); defer syscall.Close(int(fd)) }
 must(stat(fd)); assert(closed(c.Control(func(uintptr) { panic("reused descriptor callback") })))
 fmt.Println("regular errors and actual descriptor reuse", true)
}

func writes() {
 data := make([]byte,2<<20); for i := range data { data[i] = byte(i%251) }
 r,w := pipe(); c := raw(w); received := make(chan []byte); ready := make(chan struct{})
 go func() { <-ready; got,err := io.ReadAll(r); must(err); received <- got }()
 count,misses := 0,0
 must(c.Write(func(fd uintptr) bool {
  // Go waits for a new readiness event after false. Reach EAGAIN first.
  for count < len(data) {
   n,err := syscall.Write(int(fd),data[count:])
   if n > 0 { count += n }
   if err == syscall.EAGAIN { misses++; if misses == 1 { close(ready) }; return false }
   must(err)
  }
  return true
 }))
 must(w.Close()); got := <-received; must(r.Close())
 assert(misses > 0 && count == len(data) && bytes.Equal(got,data))
 fmt.Println("raw partial write transfer", true)

 r,w = pipe(); c = raw(w); entered := make(chan struct{}); done := make(chan error)
 count = 0
 go func() { first := true; done <- c.Write(func(fd uintptr) bool {
  for {
   n,err := syscall.Write(int(fd),data[count:]); if n > 0 { count += n }
   if err == syscall.EAGAIN { if first { first = false; close(entered) }; return false }
   must(err)
  }
 }) }()
 <-entered; must(w.SetWriteDeadline(time.Now().Add(-time.Second)))
 assert(errors.Is(<-done,os.ErrDeadlineExceeded)); assert(count > 0 && count < len(data))
 called := false
 err := c.Write(func(uintptr) bool { called = true; return true })
 assert(!called && errors.Is(err,os.ErrDeadlineExceeded))
 must(w.SetWriteDeadline(time.Time{}))
 entered = make(chan struct{})
 go func() { done <- c.Write(func(fd uintptr) bool {
  n,err := syscall.Write(int(fd),data[count:]); if n > 0 { count += n }
  assert(err == syscall.EAGAIN); close(entered); return false
 }) }()
 <-entered; must(w.Close()); assert(closed(<-done))
 got,err = io.ReadAll(r); must(err); must(r.Close())
 assert(len(got) == count && bytes.Equal(got,data[:count]))
 fmt.Println("raw write deadline and close byte counts", true)
}

func main() { basic(); readiness(); lifetime(); serialization(); deadlines(); regularAndReuse(); writes() }
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
