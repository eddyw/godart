package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLinuxSocketFilters(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Socket filters require Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/accept\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"syscall")
func must(err error){if err!=nil{panic(err)}}
func panics(fn func())(yes bool){defer func(){yes=recover()!=nil}();fn();return}
func result(label string,err error){if err==nil{fmt.Println(label,0);return};code,ok:=err.(syscall.Errno);if !ok{panic("non-errno")};fmt.Println(label,int(code))}
func receive(fd int,label string){data:=[]byte{31,37,41,43,47,53,59,61,67,71,73,79};n,_,err:=syscall.Recvfrom(fd,data[2:10],0);must(err);fmt.Println(label,n,data)}
func main(){
 for i,values:=range [][4]int{{0,0,0,0},{-1,-1,-1,-1},{1<<32|19,1<<40|23,1<<16|31,1<<16|37}}{fmt.Println("helpers",i,*syscall.LsfStmt(values[0],values[1]),*syscall.LsfJump(values[0],values[1],values[2],values[3]))}
 left,right:=syscall.LsfStmt(6,17),syscall.LsfStmt(6,17);right.K=19;fmt.Println("helper ownership",left!=right,left.K,right.K)
 fmt.Println("empty",panics(func(){syscall.AttachLsf(-1,nil)}),panics(func(){syscall.AttachLsf(-1,[]syscall.SockFilter{})}))
 receiver,err:=syscall.Socket(syscall.AF_INET,syscall.SOCK_DGRAM,0);must(err);must(syscall.Bind(receiver,&syscall.SockaddrInet4{Addr:[4]byte{127,0,0,1}}));must(syscall.SetsockoptTimeval(receiver,syscall.SOL_SOCKET,syscall.SO_RCVTIMEO,&syscall.Timeval{Sec:1}));target,err:=syscall.Getsockname(receiver);must(err)
 sender,err:=syscall.Socket(syscall.AF_INET,syscall.SOCK_DGRAM,0);must(err);defer syscall.Close(sender)
 program:=[]syscall.SockFilter{{Code:65535,K:31},*syscall.LsfStmt(syscall.BPF_LD|syscall.BPF_B|syscall.BPF_ABS,8),*syscall.LsfJump(syscall.BPF_JMP|syscall.BPF_JEQ|syscall.BPF_K,17,0,1),*syscall.LsfStmt(syscall.BPF_RET|syscall.BPF_K,-1),*syscall.LsfStmt(syscall.BPF_RET|syscall.BPF_K,0),{Code:65535,K:37}}
 saved:=append([]syscall.SockFilter(nil),program...);must(syscall.AttachLsf(receiver,program[1:5]));fmt.Println("caller",program[0]==saved[0],program[1]==saved[1],program[2]==saved[2],program[3]==saved[3],program[4]==saved[4],program[5]==saved[5])
 program[1].K=1;program[2].K=23;program[3].K=0
 must(syscall.Sendto(sender,[]byte{18,0,2,3,5,7,11,13},0,target));must(syscall.Sendto(sender,[]byte{17,0,2,3,5,7,11,13},0,target));data:=make([]byte,8);n,_,err:=syscall.Recvfrom(receiver,data,0);must(err);if n!=8||data[0]!=17{panic("filter did not select datagram")};fmt.Println("filter kernel copy",n,data)
 n,_,err=syscall.Recvfrom(receiver,data,syscall.MSG_DONTWAIT);fmt.Println("drop queue",n,errors.Is(err,syscall.EAGAIN))
 duplicate,err:=syscall.Dup(receiver);must(err);must(syscall.Close(receiver));result("closed attach",syscall.AttachLsf(receiver,[]syscall.SockFilter{*left}));result("closed detach",syscall.DetachLsf(receiver));must(syscall.DetachLsf(duplicate));must(syscall.Sendto(sender,[]byte{18,0,2,3,5,7,11,13},0,target));receive(duplicate,"detached bytes")
 must(syscall.AttachLsf(duplicate,[]syscall.SockFilter{*syscall.LsfStmt(syscall.BPF_RET|syscall.BPF_K,11)}));must(syscall.Sendto(sender,[]byte{17,0,2,3,5,7,11,13},0,target));receive(duplicate,"truncate")
 result("invalid code",syscall.AttachLsf(duplicate,[]syscall.SockFilter{{Code:65535,K:0}}));must(syscall.Sendto(sender,[]byte{17,0,2,3,5,7,11,13},0,target));receive(duplicate,"retained filter")
 huge:=make([]syscall.SockFilter,65536);huge[0]=*left;result("length truncation",syscall.AttachLsf(duplicate,huge));must(syscall.DetachLsf(duplicate));result("no filter",syscall.DetachLsf(duplicate));must(syscall.Close(duplicate));result("invalid attach",syscall.AttachLsf(-1,[]syscall.SockFilter{*left}));result("invalid detach",syscall.DetachLsf(-1))
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
