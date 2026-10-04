package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNativeSocketIPMreqOptions(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("Native multicast request options require Linux or macOS")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/socketipmreq\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
 "fmt"
 "syscall"
)

func get(label string,fd,level,option int) {
 value,err:=syscall.GetsockoptIPMreq(fd,level,option)
 code,typed:=err.(syscall.Errno)
 fmt.Printf("get %s nil=%t errno=%d typed=%t\n",label,value==nil,code,typed)
 if value==nil { return }
 multi,iface:=value.Multiaddr,value.Interface
 fmt.Printf("fields multi=%x interface=%x\n",multi,iface)
 value.Multiaddr[0]^=255
 value.Interface[3]^=255
 fresh,err:=syscall.GetsockoptIPMreq(fd,level,option)
 code,typed=err.(syscall.Errno)
 fmt.Printf("copy %s nil=%t errno=%d typed=%t\n",label,fresh==nil,code,typed)
 if fresh!=nil { fmt.Println("independent fields",fresh.Multiaddr==multi,fresh.Interface==iface) }
}

func set(label string,fd,level,option int,value *syscall.IPMreq) {
 var before syscall.IPMreq
 if value!=nil { before=*value }
 err:=syscall.SetsockoptIPMreq(fd,level,option,value)
 code,typed:=err.(syscall.Errno)
 fmt.Printf("set %s nil=%t unchanged=%t errno=%d typed=%t\n",label,value==nil,value==nil||*value==before,code,typed)
}

func main() {
 fd,err:=syscall.Socket(syscall.AF_INET,syscall.SOCK_DGRAM,0)
 if err!=nil { panic(err) }
 get("interface-default",fd,syscall.IPPROTO_IP,syscall.IP_MULTICAST_IF)
 get("type-partial",fd,syscall.SOL_SOCKET,syscall.SO_TYPE)
 for _,value:=range []*syscall.IPMreq{
  nil,{},
  {Multiaddr:[4]byte{127,0,0,1},Interface:[4]byte{127,0,0,1}},
  {Multiaddr:[4]byte{0,128,255,1},Interface:[4]byte{127,0,0,1}},
  {Multiaddr:[4]byte{127,0,0,1},Interface:[4]byte{203,0,113,42}},
 } {
  set("interface",fd,syscall.IPPROTO_IP,syscall.IP_MULTICAST_IF,value)
  get("interface-after",fd,syscall.IPPROTO_IP,syscall.IP_MULTICAST_IF)
 }
 group:=&syscall.IPMreq{Multiaddr:[4]byte{239,255,42,17},Interface:[4]byte{127,0,0,1}}
 set("membership-add",fd,syscall.IPPROTO_IP,syscall.IP_ADD_MEMBERSHIP,group)
 set("membership-duplicate",fd,syscall.IPPROTO_IP,syscall.IP_ADD_MEMBERSHIP,group)
 get("membership-read",fd,syscall.IPPROTO_IP,syscall.IP_ADD_MEMBERSHIP)
 set("membership-drop",fd,syscall.IPPROTO_IP,syscall.IP_DROP_MEMBERSHIP,group)
 set("membership-drop-again",fd,syscall.IPPROTO_IP,syscall.IP_DROP_MEMBERSHIP,group)
 set("membership-nil",fd,syscall.IPPROTO_IP,syscall.IP_ADD_MEMBERSHIP,nil)
 set("membership-nonmulticast",fd,syscall.IPPROTO_IP,syscall.IP_ADD_MEMBERSHIP,&syscall.IPMreq{Multiaddr:[4]byte{127,0,0,1},Interface:[4]byte{127,0,0,1}})
 set("membership-missing-interface",fd,syscall.IPPROTO_IP,syscall.IP_ADD_MEMBERSHIP,&syscall.IPMreq{Multiaddr:[4]byte{239,255,42,17},Interface:[4]byte{203,0,113,42}})
 get("unknown-option",fd,syscall.SOL_SOCKET,-1)
 get("unknown-level",fd,-1,0)
 set("unknown-option",fd,syscall.SOL_SOCKET,-1,group)
 set("unknown-level",fd,-1,0,group)
 set("unknown-option-nil",fd,syscall.SOL_SOCKET,-1,nil)
 get("wide-descriptor",fd|1<<32,syscall.SOL_SOCKET,syscall.SO_TYPE)
 get("wide-level",fd,syscall.SOL_SOCKET|1<<32,syscall.SO_TYPE)
 get("wide-option",fd,syscall.SOL_SOCKET,syscall.SO_TYPE|1<<32)
 if err:=syscall.Close(fd);err!=nil { panic(err) }
 get("closed",fd,syscall.SOL_SOCKET,syscall.SO_TYPE)
 set("closed",fd,syscall.IPPROTO_IP,syscall.IP_ADD_MEMBERSHIP,group)
 set("closed-nil",fd,syscall.IPPROTO_IP,syscall.IP_ADD_MEMBERSHIP,nil)
 get("invalid",-1,syscall.SOL_SOCKET,syscall.SO_TYPE)
 set("invalid-nil",-1,syscall.IPPROTO_IP,syscall.IP_ADD_MEMBERSHIP,nil)
 fd,err=syscall.Open("/dev/null",syscall.O_RDONLY,0)
 if err!=nil { panic(err) }
 get("file",fd,syscall.SOL_SOCKET,syscall.SO_TYPE)
 set("file",fd,syscall.IPPROTO_IP,syscall.IP_ADD_MEMBERSHIP,group)
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
