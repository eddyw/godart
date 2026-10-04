package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNativeSocketIPv6MTUInfo(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("Native IPv6 MTU info requires Linux or macOS")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/socketipv6mtu\n\ngo 1.27\n")
	fixture := `package main

import (
 "fmt"
 "syscall"
)

func get(label string,fd,level,option int) {
 value,err:=syscall.GetsockoptIPv6MTUInfo(fd,level,option)
 code,typed:=err.(syscall.Errno)
 fmt.Printf("get %s nil=%t errno=%d typed=%t\n",label,value==nil,code,typed)
 if value==nil { return }
 before:=*value
 fmt.Printf("header family=%d port=%d flow=%d addr=%x scope=%d mtu=%d\n",value.Addr.Family,value.Addr.Port,value.Addr.Flowinfo,value.Addr.Addr,value.Addr.Scope_id,value.Mtu)
 fmt.Println("length",value.Addr.Len)
 value.Addr.Family^=255
 value.Addr.Port^=0xffff
 value.Addr.Flowinfo^=0xffffffff
 value.Addr.Addr[15]^=255
 value.Addr.Scope_id^=0xffffffff
 value.Mtu^=0xffffffff
 fresh,err:=syscall.GetsockoptIPv6MTUInfo(fd,level,option)
 code,typed=err.(syscall.Errno)
 fmt.Printf("copy %s nil=%t errno=%d typed=%t\n",label,fresh==nil,code,typed)
 if fresh!=nil { fmt.Println("independent fields",*fresh==before) }
}

func main() {
 fd,err:=syscall.Socket(syscall.AF_INET6,syscall.SOCK_DGRAM,syscall.IPPROTO_ICMPV6)
 code,typed:=err.(syscall.Errno)
 fmt.Printf("icmp socket invalid=%t errno=%d typed=%t\n",fd<0,code,typed)
 if err==nil {
  for _,words:=range [][8]uint32{
   {}, {0xffffffff,0xffffffff,0xffffffff,0xffffffff,0xffffffff,0xffffffff,0xffffffff,0xffffffff},
   {0x0080ff01,0x80000000,0x7fffffff,0xffffffff,42,23,0x01020304,0x80818283},
  } {
   value:=&syscall.ICMPv6Filter{Filt:words}
   err:=syscall.SetsockoptICMPv6Filter(fd,syscall.IPPROTO_ICMPV6,syscall.ICMP6_FILTER,value)
   code,typed:=err.(syscall.Errno)
   fmt.Println("filter source",value.Filt==words,code,typed)
   get("filter-bytes",fd,syscall.IPPROTO_ICMPV6,syscall.ICMP6_FILTER)
  }
  if err:=syscall.Close(fd);err!=nil { panic(err) }
 }
 fd,err=syscall.Socket(syscall.AF_INET6,syscall.SOCK_DGRAM,0)
 if err!=nil { panic(err) }
 get("type-partial",fd,syscall.SOL_SOCKET,syscall.SO_TYPE)
 get("ipv6-only-partial",fd,syscall.IPPROTO_IPV6,syscall.IPV6_V6ONLY)
 for _,value:=range []*syscall.Timeval{{},{Sec:1},{Sec:2,Usec:100000},{}} {
  if err:=syscall.SetsockoptTimeval(fd,syscall.SOL_SOCKET,syscall.SO_RCVTIMEO,value);err!=nil { panic(err) }
  get("timeout-partial",fd,syscall.SOL_SOCKET,syscall.SO_RCVTIMEO)
 }
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
`
	if runtime.GOOS == "linux" {
		fixture = strings.ReplaceAll(fixture, ".Filt", ".Data")
		fixture = strings.ReplaceAll(fixture, "Filt:", "Data:")
		fixture = strings.ReplaceAll(fixture, "syscall.ICMP6_FILTER", "syscall.ICMPV6_FILTER")
		fixture = strings.ReplaceAll(fixture, `fmt.Println("length",value.Addr.Len)`, "")
	}
	writeFile(t, filepath.Join(dir, "main.go"), fixture)
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
