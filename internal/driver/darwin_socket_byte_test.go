package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDarwinSocketByteOptions(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Byte socket-option reads require the Darwin SDK")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/darwinsocketbyte\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
 "fmt"
 "syscall"
)

func get(label string,fd,level,option int) {
 value,err:=syscall.GetsockoptByte(fd,level,option)
 code,typed:=err.(syscall.Errno)
 fmt.Printf("get %s byte=%x errno=%d typed=%t\n",label,value,code,typed)
}

func main() {
 fd,err:=syscall.Socket(syscall.AF_INET,syscall.SOCK_DGRAM,0)
 if err!=nil { panic(err) }
 get("type",fd,syscall.SOL_SOCKET,syscall.SO_TYPE)
 get("error",fd,syscall.SOL_SOCKET,syscall.SO_ERROR)
 get("reuse",fd,syscall.SOL_SOCKET,syscall.SO_REUSEADDR)
 for _,value:=range []int{0,1,-1,-1<<31} {
  err:=syscall.SetsockoptInt(fd,syscall.SOL_SOCKET,syscall.SO_REUSEADDR,value)
  code,typed:=err.(syscall.Errno)
  fmt.Printf("set reuse value=%d errno=%d typed=%t\n",value,code,typed)
  get("reuse-after",fd,syscall.SOL_SOCKET,syscall.SO_REUSEADDR)
 }
 for _,value:=range []byte{0,1,2,127,128,255} {
  err:=syscall.SetsockoptByte(fd,syscall.IPPROTO_IP,syscall.IP_MULTICAST_TTL,value)
  code,typed:=err.(syscall.Errno)
  fmt.Printf("set ttl byte=%x errno=%d typed=%t\n",value,code,typed)
  get("ttl-after",fd,syscall.IPPROTO_IP,syscall.IP_MULTICAST_TTL)
 }
 for _,value:=range []int{257,1025,16384,65536} {
  err:=syscall.SetsockoptInt(fd,syscall.SOL_SOCKET,syscall.SO_RCVBUF,value)
  code,typed:=err.(syscall.Errno)
  fmt.Printf("set buffer value=%d errno=%d typed=%t\n",value,code,typed)
  get("buffer-after",fd,syscall.SOL_SOCKET,syscall.SO_RCVBUF)
 }
 get("linger-short",fd,syscall.SOL_SOCKET,syscall.SO_LINGER)
 get("unknown-option",fd,syscall.SOL_SOCKET,-1)
 get("unknown-level",fd,-1,0)
 get("wide-descriptor",fd|1<<32,syscall.SOL_SOCKET,syscall.SO_TYPE)
 get("wide-level",fd,syscall.SOL_SOCKET|1<<32,syscall.SO_TYPE)
 get("wide-option",fd,syscall.SOL_SOCKET,syscall.SO_TYPE|1<<32)
 if err:=syscall.Close(fd);err!=nil { panic(err) }
 get("closed",fd,syscall.SOL_SOCKET,syscall.SO_TYPE)
 get("invalid",-1,syscall.SOL_SOCKET,syscall.SO_TYPE)
 fd,err=syscall.Open("/dev/null",syscall.O_RDONLY,0)
 if err!=nil { panic(err) }
 get("file",fd,syscall.SOL_SOCKET,syscall.SO_TYPE)
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
