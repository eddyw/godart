package driver_test

import (
	"fmt"
	"github.com/eddyw/godart/internal/driver"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSyscallWebUnavailable(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("selected syscall SDK requires macOS or Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/unavailable\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"os";"syscall")
type operation struct{name string;call func()error}
func main(){
 if len(os.Args)<2{fmt.Println("native syscall declarations");return}
 calls:=[]operation{
 {"socket",func()error{_,e:=syscall.Socket(-1,syscall.SOCK_DGRAM,0);return e}},
 {"open",func()error{_,e:=syscall.Open("absent",syscall.O_RDONLY,0);return e}},
 {"seek",func()error{_,e:=syscall.Seek(-1,0,0);return e}},
 {"close",func()error{return syscall.Close(-1)}},
 {"dup",func()error{_,e:=syscall.Dup(-1);return e}},
 {"dup2",func()error{return syscall.Dup2(-1,-1)}},
 {"fsync",func()error{return syscall.Fsync(-1)}},
 {"access",func()error{return syscall.Access("absent",0)}},
 {"getrlimit",func()error{return syscall.Getrlimit(syscall.RLIMIT_NOFILE,&syscall.Rlimit{})}},
 {"setrlimit",func()error{return syscall.Setrlimit(-1,&syscall.Rlimit{})}},
 {"setnonblock",func()error{return syscall.SetNonblock(-1,true)}},
 {"readdirent",func()error{_,e:=syscall.ReadDirent(-1,make([]byte,32));return e}},
 {"getwd",func()error{_,e:=syscall.Getwd();return e}},
 {"utimes-nano",func()error{return syscall.UtimesNano("absent",[]syscall.Timespec{{},{}})}},
 {"utimes",func()error{return syscall.Utimes("absent",[]syscall.Timeval{{},{}})}},
 {"futimes",func()error{return syscall.Futimes(-1,[]syscall.Timeval{{},{}})}},
 {"unlink",func()error{return syscall.Unlink("absent")}},
 {"rmdir",func()error{return syscall.Rmdir("absent")}},
 {"readlink",func()error{_,e:=syscall.Readlink("absent",make([]byte,16));return e}},
 {"getrusage",func()error{return syscall.Getrusage(syscall.RUSAGE_SELF,&syscall.Rusage{})}},
 {"setsockopt-string",func()error{return syscall.SetsockoptString(-1,1,1,"")}},
 {"getsockopt-int",func()error{_,e:=syscall.GetsockoptInt(-1,1,1);return e}},
 {"setsockopt-int",func()error{return syscall.SetsockoptInt(-1,1,1,0)}},
 {"socketpair",func()error{_,e:=syscall.Socketpair(syscall.AF_UNIX,syscall.SOCK_DGRAM,0);return e}},
 {"getsockopt-ipmreq",func()error{_,e:=syscall.GetsockoptIPMreq(-1,1,1);return e}},
 {"setsockopt-ipmreq",func()error{return syscall.SetsockoptIPMreq(-1,1,1,&syscall.IPMreq{})}},
 {"setsockopt-linger",func()error{return syscall.SetsockoptLinger(-1,1,1,&syscall.Linger{})}},
 {"getsockopt-ipv6mreq",func()error{_,e:=syscall.GetsockoptIPv6Mreq(-1,1,1);return e}},
 {"setsockopt-ipv6mreq",func()error{return syscall.SetsockoptIPv6Mreq(-1,1,1,&syscall.IPv6Mreq{})}},
 {"setsockopt-timeval",func()error{return syscall.SetsockoptTimeval(-1,1,1,&syscall.Timeval{})}},
 {"getsockopt-icmpv6",func()error{_,e:=syscall.GetsockoptICMPv6Filter(-1,1,1);return e}},
 {"setsockopt-icmpv6",func()error{return syscall.SetsockoptICMPv6Filter(-1,1,1,&syscall.ICMPv6Filter{})}},
 {"mmap",func()error{_,e:=syscall.Mmap(-1,0,4096,syscall.PROT_READ,syscall.MAP_PRIVATE);return e}},
 {"munmap",func()error{return syscall.Munmap([]byte{1})}},
 {"shutdown",func()error{return syscall.Shutdown(-1,syscall.SHUT_RD)}},
 {"flock",func()error{return syscall.Flock(-1,syscall.LOCK_EX|syscall.LOCK_NB)}},
 {"listen",func()error{return syscall.Listen(-1,1)}},
 {"pipe",func()error{return syscall.Pipe([]int{71,73})}},
 {"bind",func()error{return syscall.Bind(-1,&syscall.SockaddrInet4{})}},
 {"connect",func()error{return syscall.Connect(-1,&syscall.SockaddrInet4{})}},
 {"sendto",func()error{return syscall.Sendto(-1,[]byte{1},0,&syscall.SockaddrInet4{})}},
 {"getsockname",func()error{_,e:=syscall.Getsockname(-1);return e}},
 {"getpeername",func()error{_,e:=syscall.Getpeername(-1);return e}},
 {"recvfrom",func()error{_,_,e:=syscall.Recvfrom(-1,make([]byte,4),0);return e}},
 {"accept",func()error{_,_,e:=syscall.Accept(-1);return e}},
 {"sendmsg",func()error{_,e:=syscall.SendmsgN(-1,[]byte{1},make([]byte,16),nil,0);return e}},
 {"recvmsg",func()error{_,_,_,_,e:=syscall.Recvmsg(-1,make([]byte,4),make([]byte,16),0);return e}},
 }
 calls=append(calls,targetCalls()...)
 valid:=true
 for _,value:=range calls{err:=value.call();code,typed:=err.(syscall.Errno);match:=typed&&code==syscall.ENOSYS&&errors.Is(err,errors.ErrUnsupported);fmt.Println(value.name,int(code),match);valid=valid&&match}
 if !valid{panic("selected SDK ENOSYS mismatch")}
 fmt.Println("all unavailable",len(calls))
}
`)
	writeFile(t, filepath.Join(dir, "target_darwin.go"), `//go:build darwin
package main
import "syscall"
func targetCalls()[]operation{return []operation{
 {"select",func()error{return syscall.Select(0,nil,nil,nil,&syscall.Timeval{})}},
 {"getsockopt-byte",func()error{_,e:=syscall.GetsockoptByte(-1,1,1);return e}},
 {"getsockopt-ipv6mtu",func()error{_,e:=syscall.GetsockoptIPv6MTUInfo(-1,1,1);return e}},
}}
`)
	writeFile(t, filepath.Join(dir, "target_linux.go"), `//go:build linux
package main
import "syscall"
func targetCalls()[]operation{return []operation{
 {"select",func()error{_,e:=syscall.Select(0,nil,nil,nil,&syscall.Timeval{});return e}},
}}
`)

	native := filepath.Join(t.TempDir(), "native")
	run(t, dir, "go", "build", "-o", native, ".")
	want := upstreamOutcome(t, dir, os.Environ(), runLimit, native)
	if want.Exit != 0 || want.Text != "native syscall declarations" {
		t.Fatalf("native declarations: %+v", want)
	}
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	names := []string{"socket", "open", "seek", "close", "dup", "dup2", "fsync", "access", "getrlimit", "setrlimit", "setnonblock", "readdirent", "getwd", "utimes-nano", "utimes", "futimes", "unlink", "rmdir", "readlink", "getrusage", "setsockopt-string", "getsockopt-int", "setsockopt-int", "socketpair", "getsockopt-ipmreq", "setsockopt-ipmreq", "setsockopt-linger", "getsockopt-ipv6mreq", "setsockopt-ipv6mreq", "setsockopt-timeval", "getsockopt-icmpv6", "setsockopt-icmpv6", "mmap", "munmap", "shutdown", "flock", "listen", "pipe", "bind", "connect", "sendto", "getsockname", "getpeername", "recvfrom", "accept", "sendmsg", "recvmsg", "select"}
	code := 38
	if runtime.GOOS == "darwin" {
		code = 78
		names = append(names, "getsockopt-byte", "getsockopt-ipv6mtu")
	}
	var lines []string
	for _, name := range names {
		lines = append(lines, fmt.Sprint(name, " ", code, " true"))
	}
	lines = append(lines, fmt.Sprint("all unavailable ", len(names)))
	expected := strings.Join(lines, "\n")
	t.Log("native syscall declarations", len(names))
	for _, target := range []string{"js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			got := runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{"unavailable"})
			if got.Exit != 0 || got.Text != expected {
				t.Fatalf("selected SDK ENOSYS differs: %+v", got)
			}
		})
	}
}
