package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNativeSocketTimevalOptions(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("Native socket timeval options require Linux or macOS")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/sockettimeval\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
 "fmt"
 "syscall"
)

func get(label string,fd,option int) {
 value,err:=syscall.GetsockoptIPv6Mreq(fd,syscall.SOL_SOCKET,option)
 code,typed:=err.(syscall.Errno)
 fmt.Printf("get %s nil=%t errno=%d typed=%t\n",label,value==nil,code,typed)
 if value!=nil { fmt.Printf("timeval bytes=%x tail=%d\n",value.Multiaddr,value.Interface) }
}

func set(label string,fd,level,option int,value *syscall.Timeval) {
 var before syscall.Timeval
 if value!=nil { before=*value }
 err:=syscall.SetsockoptTimeval(fd,level,option,value)
 code,typed:=err.(syscall.Errno)
 fmt.Printf("set %s nil=%t unchanged=%t errno=%d typed=%t\n",label,value==nil,value==nil||*value==before,code,typed)
}

func main() {
 fd,err:=syscall.Socket(syscall.AF_INET,syscall.SOCK_DGRAM,0)
 if err!=nil { panic(err) }
 for _,option:=range []int{syscall.SO_RCVTIMEO,syscall.SO_SNDTIMEO} {
  get("initial",fd,option)
  for _,value:=range []*syscall.Timeval{
   nil,{}, {Sec:1}, {Usec:100000}, {Sec:2,Usec:999999},
   {Sec:-1}, {Usec:-1}, {Usec:1000000}, {Usec:-0x80000000},
   {Sec:1<<32,Usec:0x7fffffff}, {},
  } {
   set("timeout",fd,syscall.SOL_SOCKET,option,value)
   get("after",fd,option)
  }
 }
 value:=&syscall.Timeval{Sec:3,Usec:100000}
 set("unknown-option",fd,syscall.SOL_SOCKET,-1,value)
 set("unknown-level",fd,-1,0,value)
 set("unknown-option-nil",fd,syscall.SOL_SOCKET,-1,nil)
 set("wide-descriptor",fd|1<<32,syscall.SOL_SOCKET,syscall.SO_RCVTIMEO,value)
 get("wide-descriptor-after",fd,syscall.SO_RCVTIMEO)
 set("wide-level",fd,syscall.SOL_SOCKET|1<<32,syscall.SO_RCVTIMEO,value)
 get("wide-level-after",fd,syscall.SO_RCVTIMEO)
 set("wide-option",fd,syscall.SOL_SOCKET,syscall.SO_RCVTIMEO|1<<32,value)
 get("wide-option-after",fd,syscall.SO_RCVTIMEO)
 if err:=syscall.Close(fd);err!=nil { panic(err) }
 get("closed",fd,syscall.SO_RCVTIMEO)
 set("closed",fd,syscall.SOL_SOCKET,syscall.SO_RCVTIMEO,value)
 set("closed-nil",fd,syscall.SOL_SOCKET,syscall.SO_RCVTIMEO,nil)
 set("invalid",-1,syscall.SOL_SOCKET,syscall.SO_RCVTIMEO,value)
 set("invalid-nil",-1,syscall.SOL_SOCKET,syscall.SO_RCVTIMEO,nil)
 fd,err=syscall.Open("/dev/null",syscall.O_RDONLY,0)
 if err!=nil { panic(err) }
 set("file",fd,syscall.SOL_SOCKET,syscall.SO_RCVTIMEO,value)
 if err:=syscall.Close(fd);err!=nil { panic(err) }
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
