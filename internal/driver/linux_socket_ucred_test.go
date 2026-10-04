package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLinuxSocketCredentials(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Socket credential reads require the Linux SDK")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/socketucred\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
 "fmt"
 "syscall"
)

func get(label string,fd,level,option int,owner bool) {
 value,err:=syscall.GetsockoptUcred(fd,level,option)
 code,typed:=err.(syscall.Errno)
 fmt.Printf("get %s nil=%t errno=%d typed=%t\n",label,value==nil,code,typed)
 if value==nil { return }
 pid,uid,gid:=value.Pid,value.Uid,value.Gid
 if owner { fmt.Printf("fields pid-self=%t pid-positive=%t uid=%d gid=%d uid-self=%t gid-self=%t\n",pid==int32(syscall.Getpid()),pid>0,uid,gid,uid==uint32(syscall.Getuid()),gid==uint32(syscall.Getgid())) } else { fmt.Printf("fields pid=%d uid=%d gid=%d\n",pid,uid,gid) }
 value.Pid=-123
 value.Uid=0
 value.Gid=^uint32(0)
 fresh,err:=syscall.GetsockoptUcred(fd,level,option)
 code,typed=err.(syscall.Errno)
 fmt.Printf("copy %s nil=%t errno=%d typed=%t\n",label,fresh==nil,code,typed)
 if fresh!=nil { fmt.Println("independent fields",fresh.Pid==pid,fresh.Uid==uid,fresh.Gid==gid) }
}

func main() {
 for _,kind:=range []int{syscall.SOCK_STREAM,syscall.SOCK_DGRAM,syscall.SOCK_SEQPACKET} {
  pair,err:=syscall.Socketpair(syscall.AF_UNIX,kind,0)
  if err!=nil { panic(err) }
  fmt.Println("pair type",kind)
  for _,fd:=range pair { get("peer",fd,syscall.SOL_SOCKET,syscall.SO_PEERCRED,true) }
  get("type-partial",pair[0],syscall.SOL_SOCKET,syscall.SO_TYPE,false)
  get("unknown-option",pair[0],syscall.SOL_SOCKET,-1,false)
  get("unknown-level",pair[0],-1,0,false)
  get("wide-descriptor",pair[0]|1<<32,syscall.SOL_SOCKET,syscall.SO_PEERCRED,true)
  get("wide-level",pair[0],syscall.SOL_SOCKET|1<<32,syscall.SO_PEERCRED,true)
  get("wide-option",pair[0],syscall.SOL_SOCKET,syscall.SO_PEERCRED|1<<32,true)
  if err:=syscall.Close(pair[0]);err!=nil { panic(err) }
  get("closed",pair[0],syscall.SOL_SOCKET,syscall.SO_PEERCRED,false)
  if err:=syscall.Close(pair[1]);err!=nil { panic(err) }
 }
 fd,err:=syscall.Socket(syscall.AF_INET,syscall.SOCK_DGRAM,0)
 if err!=nil { panic(err) }
 get("unconnected",fd,syscall.SOL_SOCKET,syscall.SO_PEERCRED,false)
 get("ttl-partial",fd,syscall.IPPROTO_IP,syscall.IP_TTL,false)
 if err:=syscall.SetsockoptInt(fd,syscall.IPPROTO_IP,syscall.IP_TTL,37);err!=nil { panic(err) }
 get("ttl-after",fd,syscall.IPPROTO_IP,syscall.IP_TTL,false)
 if err:=syscall.Close(fd);err!=nil { panic(err) }
 get("invalid",-1,syscall.SOL_SOCKET,syscall.SO_PEERCRED,false)
 fd,err=syscall.Open("/dev/null",syscall.O_RDONLY,0)
 if err!=nil { panic(err) }
 get("file",fd,syscall.SOL_SOCKET,syscall.SO_PEERCRED,false)
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
