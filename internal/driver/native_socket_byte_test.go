package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNativeSocketByteOptions(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("Native byte socket options require Linux or macOS")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/socketbyte\n\ngo 1.27\n")
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

func set(label string,fd,level,option int,value byte) {
 err:=syscall.SetsockoptByte(fd,level,option,value)
 code,typed:=err.(syscall.Errno)
 fmt.Printf("set %s byte=%x errno=%d typed=%t\n",label,value,code,typed)
}

func main() {
 fd,err:=syscall.Socket(syscall.AF_INET,syscall.SOCK_DGRAM,0)
 if err!=nil { panic(err) }
 for _,option:=range []struct{name string;value int}{{"multicast-ttl",syscall.IP_MULTICAST_TTL},{"multicast-loop",syscall.IP_MULTICAST_LOOP}} {
  get(option.name,fd,syscall.IPPROTO_IP,option.value)
  for _,value:=range []byte{0,1,2,127,128,255,0} {
   set(option.name,fd,syscall.IPPROTO_IP,option.value,value)
   get(option.name,fd,syscall.IPPROTO_IP,option.value)
  }
 }
 for _,value:=range []byte{0,1,255} {
  set("integer-reuse-short",fd,syscall.SOL_SOCKET,syscall.SO_REUSEADDR,value)
  get("integer-reuse-after",fd,syscall.SOL_SOCKET,syscall.SO_REUSEADDR)
  set("integer-ttl-short",fd,syscall.IPPROTO_IP,syscall.IP_TTL,value)
  get("integer-ttl-after",fd,syscall.IPPROTO_IP,syscall.IP_TTL)
 }
 set("read-only-type",fd,syscall.SOL_SOCKET,syscall.SO_TYPE,1)
 set("unknown-option",fd,syscall.SOL_SOCKET,-1,255)
 set("unknown-level",fd,-1,0,0)
 set("wide-descriptor",fd|1<<32,syscall.IPPROTO_IP,syscall.IP_MULTICAST_TTL,37)
 get("wide-descriptor-after",fd,syscall.IPPROTO_IP,syscall.IP_MULTICAST_TTL)
 set("wide-level",fd,syscall.IPPROTO_IP|1<<32,syscall.IP_MULTICAST_TTL,38)
 get("wide-level-after",fd,syscall.IPPROTO_IP,syscall.IP_MULTICAST_TTL)
 set("wide-option",fd,syscall.IPPROTO_IP,syscall.IP_MULTICAST_TTL|1<<32,39)
 get("wide-option-after",fd,syscall.IPPROTO_IP,syscall.IP_MULTICAST_TTL)
 if err:=syscall.Close(fd);err!=nil { panic(err) }
 set("closed",fd,syscall.IPPROTO_IP,syscall.IP_MULTICAST_TTL,0)
 set("invalid",-1,syscall.IPPROTO_IP,syscall.IP_MULTICAST_TTL,255)
 fd,err=syscall.Open("/dev/null",syscall.O_RDONLY,0)
 if err!=nil { panic(err) }
 set("file",fd,syscall.IPPROTO_IP,syscall.IP_MULTICAST_TTL,1)
 if err:=syscall.Close(fd);err!=nil { panic(err) }
 fd,err=syscall.Socket(syscall.AF_INET6,syscall.SOCK_DGRAM,0)
 if err!=nil { panic(err) }
 for _,value:=range []byte{0,1,255} {
  set("ipv6-integer-short",fd,syscall.IPPROTO_IPV6,syscall.IPV6_MULTICAST_LOOP,value)
  get("ipv6-integer-after",fd,syscall.IPPROTO_IPV6,syscall.IPV6_MULTICAST_LOOP)
 }
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
