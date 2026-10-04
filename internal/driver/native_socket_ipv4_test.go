package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNativeSocketIPv4Options(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("Native IPv4 socket options require Linux or macOS")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/socketipv4\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
 "fmt"
 "syscall"
)

func get(label string,fd,level,option int) {
 value,err:=syscall.GetsockoptInet4Addr(fd,level,option)
 code,typed:=err.(syscall.Errno)
 fmt.Printf("get %s bytes=%x errno=%d typed=%t\n",label,value,code,typed)
 value[0]^=255
 fresh,err:=syscall.GetsockoptInet4Addr(fd,level,option)
 code,typed=err.(syscall.Errno)
 fmt.Printf("copy %s bytes=%x errno=%d typed=%t\n",label,fresh,code,typed)
}

func set(label string,fd,level,option int,value [4]byte) {
 before:=value
 err:=syscall.SetsockoptInet4Addr(fd,level,option,value)
 code,typed:=err.(syscall.Errno)
 fmt.Printf("set %s bytes=%x unchanged=%t errno=%d typed=%t\n",label,value,value==before,code,typed)
}

func main() {
 fd,err:=syscall.Socket(syscall.AF_INET,syscall.SOCK_DGRAM,0)
 if err!=nil { panic(err) }
 get("interface-default",fd,syscall.IPPROTO_IP,syscall.IP_MULTICAST_IF)
 for _,value:=range [][4]byte{{127,0,0,1},{127,0,0,2},{203,0,113,42},{0,0,0,0}} {
  set("interface",fd,syscall.IPPROTO_IP,syscall.IP_MULTICAST_IF,value)
  get("interface-after",fd,syscall.IPPROTO_IP,syscall.IP_MULTICAST_IF)
 }
 get("type",fd,syscall.SOL_SOCKET,syscall.SO_TYPE)
 for _,value:=range [][4]byte{{0,0,0,0},{1,0,0,0},{0,0,0,128},{255,255,255,255},{0,0,0,0}} {
  set("reuse",fd,syscall.SOL_SOCKET,syscall.SO_REUSEADDR,value)
  get("reuse-after",fd,syscall.SOL_SOCKET,syscall.SO_REUSEADDR)
 }
 set("linger-short",fd,syscall.SOL_SOCKET,syscall.SO_LINGER,[4]byte{1,2,3,4})
 get("linger-after",fd,syscall.SOL_SOCKET,syscall.SO_LINGER)
 get("unknown-option",fd,syscall.SOL_SOCKET,-1)
 set("unknown-option",fd,syscall.SOL_SOCKET,-1,[4]byte{0,128,255,0})
 get("unknown-level",fd,-1,0)
 set("unknown-level",fd,-1,0,[4]byte{1,2,3,4})
 get("wide-descriptor",fd|1<<32,syscall.SOL_SOCKET,syscall.SO_TYPE)
 get("wide-level",fd,syscall.SOL_SOCKET|1<<32,syscall.SO_TYPE)
 get("wide-option",fd,syscall.SOL_SOCKET,syscall.SO_TYPE|1<<32)
 if err:=syscall.Close(fd);err!=nil { panic(err) }
 get("closed",fd,syscall.IPPROTO_IP,syscall.IP_MULTICAST_IF)
 set("closed",fd,syscall.IPPROTO_IP,syscall.IP_MULTICAST_IF,[4]byte{})
 get("invalid",-1,syscall.IPPROTO_IP,syscall.IP_MULTICAST_IF)
 set("invalid",-1,syscall.IPPROTO_IP,syscall.IP_MULTICAST_IF,[4]byte{})
 fd,err=syscall.Open("/dev/null",syscall.O_RDONLY,0)
 if err!=nil { panic(err) }
 get("file",fd,syscall.IPPROTO_IP,syscall.IP_MULTICAST_IF)
 set("file",fd,syscall.IPPROTO_IP,syscall.IP_MULTICAST_IF,[4]byte{})
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
