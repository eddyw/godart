package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNativeSocketListen(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("Native socket listen requires Linux or macOS")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/listen\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
 "fmt"
 "os"
 "syscall"
)

func errline(label string,err error) {
 code,typed:=err.(syscall.Errno)
 fmt.Println(label,code,typed)
}

func state(fd int) {
 value,err:=syscall.GetsockoptInt(fd,syscall.SOL_SOCKET,syscall.SO_ACCEPTCONN)
 fmt.Println("accept state",value)
 errline("option",err)
}

func main() {
 for _,backlog:=range []int{-1,0,1,128} {
  fd,err:=syscall.Socket(syscall.AF_INET,syscall.SOCK_STREAM,0)
  if err!=nil { panic(err) }
  fmt.Println("backlog",backlog)
  state(fd)
  errline("listen",syscall.Listen(fd,backlog))
  state(fd)
  duplicate,err:=syscall.Dup(fd)
  if err!=nil { panic(err) }
  errline("repeat",syscall.Listen(duplicate,1))
  state(fd)
  state(duplicate)
  errline("duplicate close",syscall.Close(duplicate))
  state(fd)
  errline("close",syscall.Close(fd))
 }
 fd,err:=syscall.Socket(syscall.AF_INET,syscall.SOCK_DGRAM,0)
 if err!=nil { panic(err) }
 errline("datagram",syscall.Listen(fd,1))
 state(fd)
 errline("datagram close",syscall.Close(fd))
 fds,err:=syscall.Socketpair(syscall.AF_UNIX,syscall.SOCK_STREAM,0)
 if err!=nil { panic(err) }
 errline("connected",syscall.Listen(fds[0],1))
 n,err:=syscall.Write(fds[0],[]byte{0,128,255})
 fmt.Println("retained transport",n)
 errline("write",err)
 data:=make([]byte,3)
 n,err=syscall.Read(fds[1],data)
 fmt.Println("retained bytes",n,data)
 errline("read",err)
 for _,fd:=range fds { if err:=syscall.Close(fd);err!=nil { panic(err) } }
 errline("invalid descriptor",syscall.Listen(-1,1))
 f,err:=os.CreateTemp("","listen-file")
 if err!=nil { panic(err) }
 path:=f.Name()
 defer os.Remove(path)
 errline("not socket",syscall.Listen(int(f.Fd()),1))
 errline("file close",f.Close())
}
`)
	want := runProgram(t, dir, "go", "run", ".")
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			var got programResult
			if target == "vm" {
				got = upstreamOutcome(t, dir, nil, runLimit, "dart", filepath.Join(out, "bin/main.dart"))
			} else {
				got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
			}
			if got.Exit != 0 || got.Text != want {
				t.Fatalf("%s: exit %d\nGo:\n%s\nGart:\n%s", target, got.Exit, want, got.Text)
			}
		})
	}
}
