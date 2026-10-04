package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNativeSocketIntegerOptions(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("Native integer socket options require Linux or macOS")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/socketintegers\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
 "fmt"
 "syscall"
)

func get(label string,fd,level,option int) {
 value,err:=syscall.GetsockoptInt(fd,level,option)
 code,typed:=err.(syscall.Errno)
 fmt.Printf("get %s value=%d errno=%d typed=%t\n",label,value,code,typed)
}

func set(label string,fd,level,option,value int) {
 err:=syscall.SetsockoptInt(fd,level,option,value)
 code,typed:=err.(syscall.Errno)
 fmt.Printf("set %s errno=%d typed=%t\n",label,code,typed)
}

func main() {
 for _,kind:=range []int{syscall.SOCK_STREAM,syscall.SOCK_DGRAM} {
  fd,err:=syscall.Socket(syscall.AF_INET,kind,0)
  if err!=nil { panic(err) }
  fmt.Println("socket",kind)
  for _,option:=range []struct{name string;value int}{
   {"type",syscall.SO_TYPE},{"error",syscall.SO_ERROR},{"reuse",syscall.SO_REUSEADDR},
   {"keepalive",syscall.SO_KEEPALIVE},{"broadcast",syscall.SO_BROADCAST},
   {"linger",syscall.SO_LINGER},{"receive",syscall.SO_RCVBUF},
  } { get(option.name,fd,syscall.SOL_SOCKET,option.value) }
  for _,value:=range []int{0,1,-1,1<<32,(1<<32)|1,-1<<31} {
   fmt.Println("input",value)
   set("reuse",fd,syscall.SOL_SOCKET,syscall.SO_REUSEADDR,value)
   get("reuse",fd,syscall.SOL_SOCKET,syscall.SO_REUSEADDR)
  }
  set("receive",fd,syscall.SOL_SOCKET,syscall.SO_RCVBUF,16384)
  get("receive",fd,syscall.SOL_SOCKET,syscall.SO_RCVBUF)
  set("linger-short",fd,syscall.SOL_SOCKET,syscall.SO_LINGER,1)
  get("linger-after",fd,syscall.SOL_SOCKET,syscall.SO_LINGER)
  set("type-read-only",fd,syscall.SOL_SOCKET,syscall.SO_TYPE,kind)
  get("unknown-option",fd,syscall.SOL_SOCKET,-1)
  set("unknown-option",fd,syscall.SOL_SOCKET,-1,1)
  get("unknown-level",fd,-1,0)
  set("unknown-level",fd,-1,0,1)
  get("tcp",fd,syscall.IPPROTO_TCP,syscall.TCP_NODELAY)
  set("tcp",fd,syscall.IPPROTO_TCP,syscall.TCP_NODELAY,1)
  get("tcp-after",fd,syscall.IPPROTO_TCP,syscall.TCP_NODELAY)
  get("ttl",fd,syscall.IPPROTO_IP,syscall.IP_TTL)
  set("ttl",fd,syscall.IPPROTO_IP,syscall.IP_TTL,37)
  get("ttl-after",fd,syscall.IPPROTO_IP,syscall.IP_TTL)
  set("ttl-invalid",fd,syscall.IPPROTO_IP,syscall.IP_TTL,256)
  get("ttl-after-invalid",fd,syscall.IPPROTO_IP,syscall.IP_TTL)
  get("wide-descriptor",fd|1<<32,syscall.SOL_SOCKET,syscall.SO_TYPE)
  get("wide-level",fd,syscall.SOL_SOCKET|1<<32,syscall.SO_TYPE)
  get("wide-option",fd,syscall.SOL_SOCKET,syscall.SO_TYPE|1<<32)
  if err:=syscall.Close(fd);err!=nil { panic(err) }
  get("closed",fd,syscall.SOL_SOCKET,syscall.SO_TYPE)
  set("closed",fd,syscall.SOL_SOCKET,syscall.SO_REUSEADDR,1)
 }
 get("invalid",-1,syscall.SOL_SOCKET,syscall.SO_TYPE)
 set("invalid",-1,syscall.SOL_SOCKET,syscall.SO_REUSEADDR,1)
 fd,err:=syscall.Open("/dev/null",syscall.O_RDONLY,0)
 if err!=nil { panic(err) }
 get("file",fd,syscall.SOL_SOCKET,syscall.SO_TYPE)
 set("file",fd,syscall.SOL_SOCKET,syscall.SO_REUSEADDR,1)
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
