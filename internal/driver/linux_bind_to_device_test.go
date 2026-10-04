package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLinuxBindToDevice(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("BindToDevice requires Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/accept\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"syscall")
func must(err error){if err!=nil{panic(err)}}
func socket()int{fd,err:=syscall.Socket(syscall.AF_INET,syscall.SOCK_DGRAM,0);must(err);return fd}
func device(fd int)[16]byte{value,err:=syscall.GetsockoptIPv6Mreq(fd,syscall.SOL_SOCKET,syscall.SO_BINDTODEVICE);must(err);return value.Multiaddr}
func main(){
 fmt.Println("invalid",errors.Is(syscall.BindToDevice(-1,""),syscall.EBADF),errors.Is(syscall.BindToDevice(-1,"lo\x00suffix"),syscall.EBADF))
 for i,name:=range []string{"","lo","lo\x00suffix"}{
  fd:=socket();err:=syscall.BindToDevice(fd,name)
  fmt.Println("binding",i,err==nil,errors.Is(err,syscall.EPERM),errors.Is(err,syscall.ENODEV),device(fd))
  must(syscall.Close(fd))
 }
 fd:=socket();fmt.Println("missing",errors.Is(syscall.BindToDevice(fd,"gart_missing0"),syscall.ENODEV),device(fd));must(syscall.Close(fd))
 receiver:=socket();must(syscall.BindToDevice(receiver,"lo"));must(syscall.Bind(receiver,&syscall.SockaddrInet4{Addr:[4]byte{127,0,0,1}}))
 target,err:=syscall.Getsockname(receiver);must(err);duplicate,err:=syscall.Dup(receiver);must(err)
 fmt.Println("duplicate",device(receiver)==[16]byte{'l','o'},device(duplicate)==device(receiver));must(syscall.Close(receiver))
 sender:=socket();must(syscall.BindToDevice(sender,"lo\x00suffix"));must(syscall.Sendto(sender,[]byte{0,255,17,19},0,target))
 backing:=[]byte{31,37,41,43,47,53,59,61};n,source,err:=syscall.Recvfrom(duplicate,backing[2:6],0);must(err)
 address:=source.(*syscall.SockaddrInet4)
 fmt.Println("transport",n,backing,address.Addr==[4]byte{127,0,0,1},address.Port>0)
 err=syscall.BindToDevice(duplicate,"");fmt.Println("rebind",err==nil,errors.Is(err,syscall.EPERM),device(duplicate))
 must(syscall.Close(sender));must(syscall.Close(duplicate));fmt.Println("closed",errors.Is(syscall.BindToDevice(duplicate,"lo"),syscall.EBADF))
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
