package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNativeSocketLingerOptions(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("Native socket linger options require Linux or macOS")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/socketlinger\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
 "encoding/binary"
 "fmt"
 "syscall"
)

func get(label string,fd int) {
 value,err:=syscall.GetsockoptIPMreq(fd,syscall.SOL_SOCKET,syscall.SO_LINGER)
 code,typed:=err.(syscall.Errno)
 fmt.Printf("get %s nil=%t errno=%d typed=%t\n",label,value==nil,code,typed)
 if value!=nil {
  fmt.Println("linger fields",int32(binary.NativeEndian.Uint32(value.Multiaddr[:])),int32(binary.NativeEndian.Uint32(value.Interface[:])))
 }
}

func set(label string,fd,level,option int,value *syscall.Linger) {
 var before syscall.Linger
 if value!=nil { before=*value }
 err:=syscall.SetsockoptLinger(fd,level,option,value)
 code,typed:=err.(syscall.Errno)
 fmt.Printf("set %s nil=%t unchanged=%t errno=%d typed=%t\n",label,value==nil,value==nil||*value==before,code,typed)
}

func main() {
 fd,err:=syscall.Socket(syscall.AF_INET,syscall.SOCK_STREAM,0)
 if err!=nil { panic(err) }
 get("initial",fd)
 for _,value:=range []*syscall.Linger{
  nil,{}, {Onoff:1,Linger:0}, {Onoff:1,Linger:1},
  {Onoff:1,Linger:65535}, {Onoff:1,Linger:65536},
  {Onoff:-1,Linger:-1}, {Onoff:0x7fffffff,Linger:0x7fffffff},
  {Onoff:-0x80000000,Linger:-0x80000000}, {Onoff:0,Linger:42},
 } {
  set("linger",fd,syscall.SOL_SOCKET,syscall.SO_LINGER,value)
  get("after",fd)
 }
 value:=&syscall.Linger{Onoff:1,Linger:23}
 set("unknown-option",fd,syscall.SOL_SOCKET,-1,value)
 set("unknown-level",fd,-1,0,value)
 set("unknown-option-nil",fd,syscall.SOL_SOCKET,-1,nil)
 set("wide-descriptor",fd|1<<32,syscall.SOL_SOCKET,syscall.SO_LINGER,value)
 get("wide-descriptor-after",fd)
 set("wide-level",fd,syscall.SOL_SOCKET|1<<32,syscall.SO_LINGER,value)
 get("wide-level-after",fd)
 set("wide-option",fd,syscall.SOL_SOCKET,syscall.SO_LINGER|1<<32,value)
 get("wide-option-after",fd)
 if err:=syscall.Close(fd);err!=nil { panic(err) }
 get("closed",fd)
 set("closed",fd,syscall.SOL_SOCKET,syscall.SO_LINGER,value)
 set("closed-nil",fd,syscall.SOL_SOCKET,syscall.SO_LINGER,nil)
 set("invalid",-1,syscall.SOL_SOCKET,syscall.SO_LINGER,value)
 set("invalid-nil",-1,syscall.SOL_SOCKET,syscall.SO_LINGER,nil)
 fd,err=syscall.Open("/dev/null",syscall.O_RDONLY,0)
 if err!=nil { panic(err) }
 set("file",fd,syscall.SOL_SOCKET,syscall.SO_LINGER,value)
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
