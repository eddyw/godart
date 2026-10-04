package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLinuxEpoll(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Epoll requires Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/accept\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"os";"strconv";"strings";"syscall")
func must(err error){if err!=nil{panic(err)}}
func main(){
 for _,flags:=range []int{0,syscall.EPOLL_CLOEXEC}{fd,err:=syscall.EpollCreate1(flags);must(err);data,err:=os.ReadFile(fmt.Sprintf("/proc/self/fdinfo/%d",fd));must(err);found:=false;for _,line:=range strings.Split(string(data),"\n"){if strings.HasPrefix(line,"flags:"){bits,err:=strconv.ParseUint(strings.TrimSpace(strings.TrimPrefix(line,"flags:")),8,64);must(err);fmt.Println("create flags",bits&syscall.O_CLOEXEC!=0);found=true}};if !found{panic("fdinfo flags")};must(syscall.Close(fd))}
 for _,size:=range []int{-1,0}{fd,err:=syscall.EpollCreate(size);fmt.Println("create size",fd,err==syscall.EINVAL)}
 fd,err:=syscall.EpollCreate1(1<<30);fmt.Println("create invalid",fd,err==syscall.EINVAL)
 ep,err:=syscall.EpollCreate(1);must(err);defer syscall.Close(ep)
 p:=make([]int,2);must(syscall.Pipe2(p,syscall.O_NONBLOCK));defer syscall.Close(p[0]);defer syscall.Close(p[1])
 marker:=syscall.EpollEvent{Events:0x12345678,Fd:-17,Pad:-31};events:=[]syscall.EpollEvent{marker,marker,marker,marker};view:=events[1:3]
 n,err:=syscall.EpollWait(ep,view,0);must(err);fmt.Println("empty",n,events[0]==marker,events[1]==marker,events[2]==marker,events[3]==marker)
 entry:=&syscall.EpollEvent{Events:syscall.EPOLLIN,Fd:int32(p[0]),Pad:-123}
 must(syscall.EpollCtl(ep,syscall.EPOLL_CTL_ADD,p[0],entry));fmt.Println("entry retained",entry.Events==syscall.EPOLLIN,entry.Fd==int32(p[0]),entry.Pad==-123)
 err=syscall.EpollCtl(ep,syscall.EPOLL_CTL_ADD,p[0],entry);fmt.Println("duplicate add",err==syscall.EEXIST)
 w,err:=syscall.Write(p[1],[]byte("ready"));must(err);if w!=5{panic("write")}
 for i:=0;i<2;i++{n,err=syscall.EpollWait(ep,view,0);must(err);fmt.Println("level",n,view[0].Events==syscall.EPOLLIN,view[0].Fd==int32(p[0]),view[0].Pad==-123,events[0]==marker,events[2]==marker,events[3]==marker)}
 entry.Events=syscall.EPOLLIN|syscall.EPOLLONESHOT;entry.Pad=-2147483648;must(syscall.EpollCtl(ep,syscall.EPOLL_CTL_MOD,p[0],entry))
 n,err=syscall.EpollWait(ep,view,0);must(err);fmt.Println("oneshot",n,view[0].Events==syscall.EPOLLIN,view[0].Pad==entry.Pad)
 n,err=syscall.EpollWait(ep,view,0);must(err);fmt.Println("disabled",n)
 must(syscall.EpollCtl(ep,syscall.EPOLL_CTL_MOD,p[0],entry));n,err=syscall.EpollWait(ep,view,0);must(err);fmt.Println("rearm",n)
 b:=make([]byte,5);r,err:=syscall.Read(p[0],b);must(err);fmt.Println("bytes",r,string(b))
 must(syscall.EpollCtl(ep,syscall.EPOLL_CTL_DEL,p[0],nil));err=syscall.EpollCtl(ep,syscall.EPOLL_CTL_DEL,p[0],nil);fmt.Println("missing delete",err==syscall.ENOENT)
 err=syscall.EpollCtl(ep,syscall.EPOLL_CTL_ADD,p[0],nil);fmt.Println("nil add",err==syscall.EFAULT)
 err=syscall.EpollCtl(ep,1<<30,p[0],entry);fmt.Println("bad operation",err==syscall.EINVAL)
 n,err=syscall.EpollWait(ep,nil,0);fmt.Println("nil wait",n,err==syscall.EINVAL)
 n,err=syscall.EpollWait(ep,[]syscall.EpollEvent{},0);fmt.Println("empty wait",n,err==syscall.EINVAL)
 entry.Events=syscall.EPOLLIN;must(syscall.EpollCtl(ep,syscall.EPOLL_CTL_ADD,p[0],entry))
 duplicate,err:=syscall.Dup(ep);must(err);defer syscall.Close(duplicate);must(syscall.Close(ep));must(syscall.Close(p[1]))
 n,err=syscall.EpollWait(duplicate,view,0);must(err);fmt.Println("duplicate HUP",n,view[0].Events&syscall.EPOLLHUP!=0,view[0].Fd==int32(p[0]),view[0].Pad==entry.Pad)
 n,err=syscall.EpollWait(ep,view,0);fmt.Println("closed",n,err==syscall.EBADF)
 err=syscall.EpollCtl(ep,syscall.EPOLL_CTL_DEL,p[0],nil);fmt.Println("closed control",err==syscall.EBADF)
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
