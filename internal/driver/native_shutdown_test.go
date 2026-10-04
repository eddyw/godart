package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNativeSocketShutdown(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("Native socket shutdown requires Linux or macOS")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/shutdown\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
 "fmt"
 "os"
 "syscall"
)

func errline(label string, err error) {
 code,typed:=err.(syscall.Errno)
 fmt.Println(label,code,typed)
}

func read(label string, fd int) {
 data:=make([]byte,8)
 n,err:=syscall.Read(fd,data)
 fmt.Println(label,n)
 errline("read",err)
 if n>0 { fmt.Println("bytes",data[:n]) }
}

func pair() [2]int {
 fds,err:=syscall.Socketpair(syscall.AF_UNIX,syscall.SOCK_STREAM,0)
 if err!=nil { panic(err) }
 for _,fd:=range fds { if err:=syscall.SetNonblock(fd,true);err!=nil { panic(err) } }
 return fds
}

func main() {
 for _,how:=range []int{syscall.SHUT_RD,syscall.SHUT_WR,syscall.SHUT_RDWR} {
  fds:=pair()
  fmt.Println("direction",how)
  n,err:=syscall.Write(fds[0],[]byte{0,128,255})
  fmt.Println("queued",n)
  errline("write",err)
  errline("shutdown",syscall.Shutdown(fds[0],how))
  read("peer queued",fds[1])
  read("peer remainder",fds[1])
  if how==syscall.SHUT_WR {
   n,err:=syscall.Write(fds[1],[]byte{42,0,200})
   fmt.Println("reverse",n)
   errline("write",err)
  }
  read("local state",fds[0])
  errline("repeat",syscall.Shutdown(fds[0],how))
  for _,fd:=range fds {
   value,err:=syscall.GetsockoptInt(fd,syscall.SOL_SOCKET,syscall.SO_TYPE)
   fmt.Println("still owned",value==syscall.SOCK_STREAM)
   errline("option",err)
   errline("close",syscall.Close(fd))
  }
 }
 fds:=pair()
 duplicate,err:=syscall.Dup(fds[0])
 if err!=nil { panic(err) }
 errline("duplicate shutdown",syscall.Shutdown(duplicate,syscall.SHUT_WR))
 read("duplicate peer EOF",fds[1])
 errline("duplicate close",syscall.Close(duplicate))
 value,err:=syscall.GetsockoptInt(fds[0],syscall.SOL_SOCKET,syscall.SO_TYPE)
 fmt.Println("original owned",value==syscall.SOCK_STREAM)
 errline("option",err)
 for _,how:=range []int{-1,3} { errline("invalid direction",syscall.Shutdown(fds[0],how)) }
 for _,fd:=range fds { if err:=syscall.Close(fd);err!=nil { panic(err) } }
 errline("invalid descriptor",syscall.Shutdown(-1,syscall.SHUT_RDWR))
 f,err:=os.CreateTemp("","shutdown-file")
 if err!=nil { panic(err) }
 path:=f.Name()
 defer os.Remove(path)
 fd:=int(f.Fd())
 errline("not socket",syscall.Shutdown(fd,syscall.SHUT_RDWR))
 errline("file close",f.Close())
 s,err:=syscall.Socket(syscall.AF_INET,syscall.SOCK_STREAM,0)
 if err!=nil { panic(err) }
 errline("unconnected",syscall.Shutdown(s,syscall.SHUT_RDWR))
 errline("socket close",syscall.Close(s))
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
