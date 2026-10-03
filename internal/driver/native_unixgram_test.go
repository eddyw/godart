package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNativeUnixDatagramContracts(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("Unix datagrams require native descriptors")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/unixgramcontracts\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("bytes";"errors";"fmt";"net";"os";"path/filepath";"runtime";"syscall";"time")
func require(ok bool, message string){if !ok{panic(message)}}
func main(){
 dir,err:=os.MkdirTemp("","ug");if err!=nil{panic(err)};defer os.RemoveAll(dir)
 serverName,clientName:=filepath.Join(dir,"s"),filepath.Join(dir,"c")
 server,err:=net.ListenUnixgram("unixgram",&net.UnixAddr{Name:serverName,Net:"unixgram"});if err!=nil{panic(err)};defer server.Close()
 client,err:=net.DialUnix("unixgram",&net.UnixAddr{Name:clientName,Net:"unixgram"},server.LocalAddr().(*net.UnixAddr));if err!=nil{panic(err)};defer client.Close()
 require(server.SetReadBuffer(8192)==nil,"read buffer");require(client.SetWriteBuffer(8192)==nil,"write buffer")
 payload:=[]byte{0,255,1,128,2};n,err:=client.Write(payload);require(err==nil&&n==len(payload),"write")
 buffer:=make([]byte,64);n,from,err:=server.ReadFromUnix(buffer);require(err==nil&&bytes.Equal(buffer[:n],payload)&&from.Name==clientName,"sender bytes")
 n,err=server.WriteToUnix([]byte("reply"),from);require(err==nil&&n==5,"reply write");n,err=client.Read(buffer);require(err==nil&&string(buffer[:n])=="reply","reply read")
 _,err=client.Write([]byte("truncated"));require(err==nil,"truncate write");n,_,flags,_,err:=server.ReadMsgUnix(buffer[:2],nil);require(err==nil&&n==2&&flags&syscall.MSG_TRUNC!=0&&string(buffer[:2])=="tr","truncate read")
 require(server.SetReadDeadline(time.Unix(0,0))==nil,"epoch");_,_,err=server.ReadFromUnix(buffer);require(errors.Is(err,os.ErrDeadlineExceeded),"epoch timeout")
 pending:=make(chan error,1);require(server.SetReadDeadline(time.Time{})==nil,"cancel deadline")
 go func(){_,_,err:=server.ReadFromUnix(buffer);pending<-err}();time.Sleep(20*time.Millisecond)
 require(server.SetReadDeadline(time.Now().Add(-time.Second))==nil,"pending deadline");require(errors.Is(<-pending,os.ErrDeadlineExceeded),"pending timeout")
 require(server.SetReadDeadline(time.Now().Add(20*time.Millisecond))==nil,"short deadline")
 go func(){_,_,err:=server.ReadFromUnix(buffer);pending<-err}();time.Sleep(5*time.Millisecond);require(server.SetReadDeadline(time.Time{})==nil,"pending cancel");time.Sleep(25*time.Millisecond)
 _,err=client.Write([]byte("cancelled"));require(err==nil,"cancel write");require(<-pending==nil&&string(buffer[:9])=="cancelled","cancel read")
 require(client.SetWriteDeadline(time.Unix(0,0))==nil,"write epoch");_,err=client.Write(payload);require(errors.Is(err,os.ErrDeadlineExceeded),"write timeout");require(client.SetWriteDeadline(time.Time{})==nil,"write cancel")
 fileName:=filepath.Join(dir,"rights");require(os.WriteFile(fileName,[]byte("native descriptor data"),0600)==nil,"file write")
 file,err:=os.Open(fileName);if err!=nil{panic(err)}
 control:=syscall.UnixRights(int(file.Fd()))
 n,oobn,err:=server.WriteMsgUnix([]byte("rights"),control,client.LocalAddr().(*net.UnixAddr));if err!=nil||n!=6||oobn!=len(control){panic(fmt.Sprintf("rights write: n=%d oobn=%d error=%v",n,oobn,err))};require(file.Close()==nil,"sender ownership")
 received:=make([]byte,64);n,oobn,flags,from,err=client.ReadMsgUnix(buffer,received);require(err==nil&&n==6&&oobn>=syscall.CmsgLen(4)&&flags&syscall.MSG_CTRUNC==0&&from.Name==serverName&&string(buffer[:n])=="rights","rights read")
 messages,err:=syscall.ParseSocketControlMessage(received[:oobn]);require(err==nil&&len(messages)==1,"rights messages")
 require(messages[0].Header.Level==syscall.SOL_SOCKET&&messages[0].Header.Type==syscall.SCM_RIGHTS,"rights header")
 fds,err:=syscall.ParseUnixRights(&messages[0]);require(err==nil&&len(fds)==1,"rights descriptors")
 fd:=fds[0];adopted:=os.NewFile(uintptr(fd),"received");require(adopted!=nil,"received file");data,err:=os.ReadFile(fileName);require(err==nil,"original file")
 got:=make([]byte,len(data));n,err=adopted.Read(got);require(err==nil&&bytes.Equal(got[:n],data),"descriptor contents");require(adopted.Close()==nil&&errors.Is(syscall.Close(fd),syscall.EBADF),"receiver ownership")
 file,err=os.Open(fileName);if err!=nil{panic(err)};control=syscall.UnixRights(int(file.Fd()))
 n,oobn,err=server.WriteMsgUnix(nil,control,client.LocalAddr().(*net.UnixAddr));require(err==nil&&n==0&&oobn==len(control),"zero-data rights write");require(file.Close()==nil,"zero-data sender ownership")
 n,oobn,_,_,err=client.ReadMsgUnix(nil,received);expected:=0;if runtime.GOOS=="darwin"{expected=1};require(err==nil&&n==expected&&oobn>=syscall.CmsgLen(4),"zero-data rights read")
 messages,err=syscall.ParseSocketControlMessage(received[:oobn]);require(err==nil&&len(messages)==1,"zero-data messages")
 fds,err=syscall.ParseUnixRights(&messages[0]);require(err==nil&&len(fds)==1,"zero-data descriptors");fd=fds[0];require(syscall.Close(fd)==nil,"zero-data receiver ownership")
 malformed:=syscall.UnixRights();malformed[0]=0;_,err=syscall.ParseSocketControlMessage(malformed);require(errors.Is(err,syscall.EINVAL),"short header length")
 for i:=0;i<syscall.SizeofCmsghdr-8;i++{malformed[i]=255};_,err=syscall.ParseSocketControlMessage(malformed);require(errors.Is(err,syscall.EINVAL),"oversized header length")
 wrong:=syscall.SocketControlMessage{Header:syscall.Cmsghdr{Level:0,Type:syscall.SCM_RIGHTS},Data:[]byte{0,0,0,0}};_,err=syscall.ParseUnixRights(&wrong);require(errors.Is(err,syscall.EINVAL),"wrong control level")
 wrong.Header.Level=syscall.SOL_SOCKET;wrong.Header.Type=-1;_,err=syscall.ParseUnixRights(&wrong);require(errors.Is(err,syscall.EINVAL),"wrong control type")
 for _,zero:=range []*net.UnixConn{new(net.UnixConn)} {
  _,_,_,_,err=zero.ReadMsgUnix(nil,nil);require(errors.Is(err,syscall.EINVAL),"zero readmsg")
  _,_,err=zero.WriteMsgUnix(nil,nil,nil);require(errors.Is(err,syscall.EINVAL),"zero writemsg")
  _,err=zero.WriteToUnix(nil,nil);require(errors.Is(err,syscall.EINVAL),"zero writeto")
  _,err=zero.WriteTo(nil,nil);require(errors.Is(err,syscall.EINVAL),"zero generic writeto")
 }
 closing,err:=net.ListenUnixgram("unixgram",&net.UnixAddr{Name:filepath.Join(dir,"closing"),Net:"unixgram"});if err!=nil{panic(err)}
 go func(){_,_,err:=closing.ReadFromUnix(buffer);pending<-err}();time.Sleep(10*time.Millisecond);require(closing.Close()==nil,"close");require(errors.Is(<-pending,net.ErrClosed),"pending close")
 replacement,err:=net.ListenUnixgram("unixgram",&net.UnixAddr{Name:filepath.Join(dir,"replacement"),Net:"unixgram"});if err!=nil{panic(err)};defer replacement.Close()
 _,err=closing.WriteToUnix(payload,replacement.LocalAddr().(*net.UnixAddr));require(errors.Is(err,net.ErrClosed),"closed handle reuse")
 blocked,err:=net.ListenUnixgram("unixgram",&net.UnixAddr{Name:filepath.Join(dir,"blocked"),Net:"unixgram"});if err!=nil{panic(err)};defer blocked.Close()
 writer,err:=net.DialUnix("unixgram",nil,blocked.LocalAddr().(*net.UnixAddr));if err!=nil{panic(err)};defer writer.Close()
 writes:=make(chan int,1)
 go func(){count:=0;for{_,err:=writer.Write(make([]byte,1024));if err!=nil{writes<-count;pending<-err;return};count++}}()
 time.Sleep(30*time.Millisecond);require(writer.SetWriteDeadline(time.Now().Add(-time.Second))==nil,"pending write deadline");count,writeErr:=<-writes,<-pending;if count==0||(runtime.GOOS=="darwin"&&!errors.Is(writeErr,syscall.ENOBUFS))||(runtime.GOOS=="linux"&&!errors.Is(writeErr,os.ErrDeadlineExceeded)){panic(fmt.Sprintf("backpressure count=%d error=%v",count,writeErr))}
 _,_,err=client.WriteMsgUnix(nil,nil,nil);require(errors.Is(err,net.ErrWriteToConnected),"connected nil message")
 _,err=server.WriteToUnix(payload,&net.UnixAddr{Name:clientName,Net:"unix"});require(errors.Is(err,syscall.EAFNOSUPPORT),"destination network")
 fmt.Println("Unix datagram contracts pass")
}
`)
	want := programOutcome(t, dir, "go", "run", ".")
	if want.Exit != 0 {
		t.Fatalf("native Go failed: %+v", want)
	}
	err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)})
	if err != nil {
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
