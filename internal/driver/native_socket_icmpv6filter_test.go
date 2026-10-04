package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNativeSocketICMPv6FilterOptions(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("Native ICMPv6 filter options require Linux or macOS")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/socketicmpv6filter\n\ngo 1.27\n")
	fixture := `package main

import (
 "fmt"
 "syscall"
)

func get(label string,fd,level,option int) {
 value,err:=syscall.GetsockoptICMPv6Filter(fd,level,option)
 code,typed:=err.(syscall.Errno)
 fmt.Printf("get %s nil=%t errno=%d typed=%t\n",label,value==nil,code,typed)
 if value==nil { return }
 words:=value.Filt
 fmt.Printf("filter words=%x\n",words)
 for i:=range value.Filt { value.Filt[i]^=0xffffffff }
 fresh,err:=syscall.GetsockoptICMPv6Filter(fd,level,option)
 code,typed=err.(syscall.Errno)
 fmt.Printf("copy %s nil=%t errno=%d typed=%t\n",label,fresh==nil,code,typed)
 if fresh!=nil { fmt.Println("independent fields",fresh.Filt==words) }
}

func set(label string,fd,level,option int,value *syscall.ICMPv6Filter) {
 var before syscall.ICMPv6Filter
 if value!=nil { before=*value }
 err:=syscall.SetsockoptICMPv6Filter(fd,level,option,value)
 code,typed:=err.(syscall.Errno)
 fmt.Printf("set %s nil=%t unchanged=%t errno=%d typed=%t\n",label,value==nil,value==nil||*value==before,code,typed)
}

func main() {
 for _,kind:=range []int{syscall.SOCK_DGRAM,syscall.SOCK_RAW} {
  fd,err:=syscall.Socket(syscall.AF_INET6,kind,syscall.IPPROTO_ICMPV6)
  code,typed:=err.(syscall.Errno)
  fmt.Printf("socket kind=%d invalid=%t errno=%d typed=%t\n",kind,fd<0,code,typed)
  if err!=nil { continue }
  get("filter-initial",fd,syscall.IPPROTO_ICMPV6,syscall.ICMP6_FILTER)
  for _,value:=range []*syscall.ICMPv6Filter{
   nil,{},
   {Filt:[8]uint32{0xffffffff,0xffffffff,0xffffffff,0xffffffff,0xffffffff,0xffffffff,0xffffffff,0xffffffff}},
   {Filt:[8]uint32{0,1,0x80000000,0x7fffffff,0xffffffff,0x0080ff01,42,23}},
  } {
   set("filter",fd,syscall.IPPROTO_ICMPV6,syscall.ICMP6_FILTER,value)
   get("filter-after",fd,syscall.IPPROTO_ICMPV6,syscall.ICMP6_FILTER)
  }
  if err:=syscall.Close(fd);err!=nil { panic(err) }
 }
 fd,err:=syscall.Socket(syscall.AF_INET6,syscall.SOCK_DGRAM,0)
 if err!=nil { panic(err) }
 get("type-partial",fd,syscall.SOL_SOCKET,syscall.SO_TYPE)
 get("timeout-initial",fd,syscall.SOL_SOCKET,syscall.SO_RCVTIMEO)
 for _,value:=range []*syscall.ICMPv6Filter{
  nil,{}, {Filt:[8]uint32{1}}, {Filt:[8]uint32{2,0,100000}},
  {Filt:[8]uint32{0xffffffff,0xffffffff,0xffffffff,0xffffffff,0xffffffff,0xffffffff,0xffffffff,0xffffffff}},
 } {
  set("timeout",fd,syscall.SOL_SOCKET,syscall.SO_RCVTIMEO,value)
  get("timeout-after",fd,syscall.SOL_SOCKET,syscall.SO_RCVTIMEO)
 }
 value:=&syscall.ICMPv6Filter{Filt:[8]uint32{1,2,3,4,5,6,7,8}}
 get("unknown-option",fd,syscall.SOL_SOCKET,-1)
 get("unknown-level",fd,-1,0)
 set("unknown-option",fd,syscall.SOL_SOCKET,-1,value)
 set("unknown-level",fd,-1,0,value)
 set("unknown-option-nil",fd,syscall.SOL_SOCKET,-1,nil)
 get("wide-descriptor",fd|1<<32,syscall.SOL_SOCKET,syscall.SO_TYPE)
 get("wide-level",fd,syscall.SOL_SOCKET|1<<32,syscall.SO_TYPE)
 get("wide-option",fd,syscall.SOL_SOCKET,syscall.SO_TYPE|1<<32)
 if err:=syscall.Close(fd);err!=nil { panic(err) }
 get("closed",fd,syscall.SOL_SOCKET,syscall.SO_TYPE)
 set("closed",fd,syscall.SOL_SOCKET,syscall.SO_RCVTIMEO,value)
 set("closed-nil",fd,syscall.SOL_SOCKET,syscall.SO_RCVTIMEO,nil)
 get("invalid",-1,syscall.SOL_SOCKET,syscall.SO_TYPE)
 set("invalid-nil",-1,syscall.SOL_SOCKET,syscall.SO_RCVTIMEO,nil)
 fd,err=syscall.Open("/dev/null",syscall.O_RDONLY,0)
 if err!=nil { panic(err) }
 get("file",fd,syscall.SOL_SOCKET,syscall.SO_TYPE)
 set("file",fd,syscall.SOL_SOCKET,syscall.SO_RCVTIMEO,value)
 if err:=syscall.Close(fd);err!=nil { panic(err) }
}
`
	if runtime.GOOS == "linux" {
		fixture = strings.ReplaceAll(fixture, ".Filt", ".Data")
		fixture = strings.ReplaceAll(fixture, "Filt:", "Data:")
		fixture = strings.ReplaceAll(fixture, "syscall.ICMP6_FILTER", "syscall.ICMPV6_FILTER")
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
