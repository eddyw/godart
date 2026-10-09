package driver_test

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestNativeTCPListenerFiles(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native TCP listener files require macOS or Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/listenerfiles\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("bytes";"errors";"fmt";"io";"net";"os";"syscall";"time")
func must(err error){if err!=nil{panic(err)}}
func count()int{entries,err:=os.ReadDir("/dev/fd");must(err);return len(entries)}
func errorOp(err error)string{var e *os.SyscallError;if errors.As(err,&e){return e.Syscall};return ""}
func exercise(network,address string){
 first,err:=net.Listen(network,address);must(err);original:=first.(*net.TCPListener)
 file,err:=original.File();must(err);other,err:=original.File();must(err)
 fmt.Println(network,"copies",file.Fd()!=other.Fd(),descriptorFlags(int(file.Fd()),syscall.F_GETFD)&syscall.FD_CLOEXEC!=0);must(other.Close())
 copied,err:=net.FileListener(file);must(err);fmt.Println(network,"address",copied.Addr().String()==original.Addr().String(),file.Name()==network+":"+original.Addr().String()+"->")
 must(original.Close());must(file.Close());_,err=original.File();fmt.Println(network,"closed",errors.Is(err,net.ErrClosed))
 clientDone:=make(chan bool,1)
 payload:=bytes.Repeat([]byte{0,128,255},1<<16)
 go func(){client,err:=net.Dial(network,copied.Addr().String());must(err);defer client.Close();_,err=client.Write([]byte("x"));must(err);got:=make([]byte,len(payload));_,err=io.ReadFull(client,got);must(err);end:=make([]byte,1);n,err:=client.Read(end);clientDone<-bytes.Equal(got,payload)&&n==0&&err==io.EOF}()
 accepted,err:=copied.Accept();must(err);socket:=accepted.(*net.TCPConn)
 must(socket.SetNoDelay(false));must(socket.SetNoDelay(true));must(socket.SetKeepAlive(true));must(socket.SetKeepAlive(false));must(socket.SetLinger(-1));must(socket.SetReadBuffer(4096));must(socket.SetWriteBuffer(4096))
 must(socket.SetReadDeadline(time.Now().Add(-time.Second)));one:=make([]byte,1);_,err=socket.Read(one);fmt.Println(network,"deadline",errors.Is(err,os.ErrDeadlineExceeded));must(socket.SetReadDeadline(time.Time{}));_,err=io.ReadFull(socket,one);must(err)
 must(copied.Close());n,err:=socket.Write(payload);must(err);must(socket.CloseWrite());fmt.Println(network,"transport",n==len(payload),one[0]=='x',<-clientDone);must(socket.Close())
 listener,err:=net.Listen(network,address);must(err);done:=make(chan error,1);go func(){_,err:=listener.Accept();done<-err}();time.Sleep(20*time.Millisecond);must(listener.Close());fmt.Println(network,"accept close",errors.Is(<-done,net.ErrClosed))
 resetListener,err:=net.Listen(network,address);must(err)
 resetClient,err:=net.Dial(network,resetListener.Addr().String());must(err)
 resetServer,err:=resetListener.Accept();must(err);must(resetClient.(*net.TCPConn).SetLinger(0));must(resetClient.Close())
 _,err=resetServer.Read(one);must(resetServer.SetDeadline(time.Now().Add(time.Second)))
 writeError:=resetServer.(*net.TCPConn).CloseWrite();readError:=resetServer.(*net.TCPConn).CloseRead()
 fmt.Println(network,"reset shutdown",err!=nil,writeError!=nil,readError!=nil)
 must(resetServer.Close());must(resetListener.Close())
}
func main(){
 exercise("tcp4","127.0.0.1:0");exercise("tcp6","[::1]:0")
 before:=count();all:=true
 for i:=0;i<8;i++{listener,err:=net.Listen("tcp4","127.0.0.1:0");must(err);file,err:=listener.(*net.TCPListener).File();must(err);adopted,err:=net.FileListener(file);must(err);must(listener.Close());must(adopted.Close());must(file.Close())}
 fmt.Println("repeat cleanup",all,count()==before)
 file,err:=os.CreateTemp("","not-listener-");must(err);defer os.Remove(file.Name());defer file.Close();before=count()
 for i:=0;i<8;i++{listener,err:=net.FileListener(file);all=all&&listener==nil&&errors.Is(err,syscall.ENOTSOCK)&&errorOp(err)=="getsockopt"}
 fmt.Println("invalid cleanup",all,count()==before)
 fd,err:=syscall.Socket(syscall.AF_INET,syscall.SOCK_DGRAM,0);must(err);datagram:=os.NewFile(uintptr(fd),"datagram");before=count();listener,err:=net.FileListener(datagram);fmt.Println("datagram source",listener==nil,errors.Is(err,syscall.EINVAL),count()==before,errorOp(err)=="");must(datagram.Close())
 closed,err:=net.Listen("tcp4","127.0.0.1:0");must(err);source,err:=closed.(*net.TCPListener).File();must(err);must(source.Close());must(closed.Close());listener,err=net.FileListener(source);fmt.Println("closed source",listener==nil,errors.Is(err,syscall.EBADF),errorOp(err)=="fcntl")
 var zero net.TCPListener;_,err=zero.File();fmt.Println("zero listener",errors.Is(err,syscall.EINVAL))
}
`)
	writeFile(t, filepath.Join(dir, "flags_native.go"), `//go:build !dart
package main
import "syscall"
func descriptorFlags(fd,cmd int)int{flags,_,code:=syscall.Syscall(syscall.SYS_FCNTL,uintptr(fd),uintptr(cmd),0);if code!=0{panic(code)};return int(flags)}
`)
	writeFile(t, filepath.Join(dir, "flags_dart.go"), `//go:build dart
package main
import "syscall/dart"
func descriptorFlags(fd,cmd int)int{flags,code:=dart.SyscallFcntl(fd,cmd,0);if code!=0{panic(code)};return flags}
`)
	want := programOutcome(t, dir, "go", "run", ".")
	const expected = "tcp4 copies true true\ntcp4 address true true\ntcp4 closed true\ntcp4 deadline true\ntcp4 transport true true true\ntcp4 accept close true\ntcp4 reset shutdown true true true\ntcp6 copies true true\ntcp6 address true true\ntcp6 closed true\ntcp6 deadline true\ntcp6 transport true true true\ntcp6 accept close true\ntcp6 reset shutdown true true true\nrepeat cleanup true true\ninvalid cleanup true true\ndatagram source true true true true\nclosed source true true true\nzero listener true"
	if want.Exit != 0 || strings.TrimSpace(want.Text) != expected {
		t.Fatalf("native TCP listener contract: %+v", want)
	}
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			var got programResult
			if target == "vm" {
				got = programOutcome(t, dir, "dart", filepath.Join(out, "bin/main.dart"))
			} else {
				got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
			}
			if got != want {
				t.Fatalf("Go:\n%s\nGart (exit %d):\n%s", want.Text, got.Exit, got.Text)
			}
		})
	}
}
