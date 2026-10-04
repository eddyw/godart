package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLinuxCredentialCodecs(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Credential codecs require Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/accept\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("bytes";"errors";"fmt";"syscall")
func must(err error){if err!=nil{panic(err)}}
func panics(fn func())(yes bool){defer func(){yes=recover()!=nil}();fn();return}
func main(){
 fmt.Println("nil",panics(func(){syscall.UnixCredentials(nil)}),panics(func(){syscall.ParseUnixCredentials(nil)}))
 for i,value:=range []syscall.Ucred{{},{Pid:-1,Uid:1<<32-1,Gid:1<<31|19},{Pid:-1<<31,Uid:17,Gid:23},{Pid:1<<31-1,Uid:31,Gid:37}}{
  saved:=value;wire:=syscall.UnixCredentials(&value);fmt.Println("wire",i,len(wire),wire);messages,err:=syscall.ParseSocketControlMessage(wire);must(err);if len(messages)!=1{panic("message count")};message:=messages[0];parsed,err:=syscall.ParseUnixCredentials(&message);must(err)
  fmt.Println("roundtrip",message.Header.Len==uint64(syscall.CmsgLen(syscall.SizeofUcred)),message.Header.Level==syscall.SOL_SOCKET,message.Header.Type==syscall.SCM_CREDENTIALS,*parsed==saved)
  savedWire:=append([]byte(nil),wire...);value.Pid=41;value.Uid=43;value.Gid=47;fmt.Println("encoder copy",bytes.Equal(wire,savedWire));old:=*parsed;message.Data[0]^=255;fmt.Println("parser copy",*parsed==old);savedData:=append([]byte(nil),message.Data...);parsed.Uid=43;fmt.Println("return copy",bytes.Equal(message.Data,savedData))
 }
 for size:=0;size<12;size++{message:=syscall.SocketControlMessage{Data:make([]byte,size)};message.Header.Level=syscall.SOL_SOCKET;message.Header.Type=syscall.SCM_CREDENTIALS;value,err:=syscall.ParseUnixCredentials(&message);fmt.Println("short",size,value==nil,errors.Is(err,syscall.EINVAL))}
 message:=syscall.SocketControlMessage{Data:[]byte{1,0,0,0,2,0,0,0,3,0,0,0,255,255}};message.Header.Level=-1;message.Header.Type=syscall.SCM_CREDENTIALS;value,err:=syscall.ParseUnixCredentials(&message);fmt.Println("level",value==nil,errors.Is(err,syscall.EINVAL));message.Header.Level=syscall.SOL_SOCKET;message.Header.Type=-1;value,err=syscall.ParseUnixCredentials(&message);fmt.Println("type",value==nil,errors.Is(err,syscall.EINVAL));message.Header.Type=syscall.SCM_CREDENTIALS;value,err=syscall.ParseUnixCredentials(&message);must(err);fmt.Println("extra data zero header",*value,message.Data)
 pair,err:=syscall.Socketpair(syscall.AF_UNIX,syscall.SOCK_DGRAM,0);must(err);defer syscall.Close(pair[0]);defer syscall.Close(pair[1]);must(syscall.SetsockoptInt(pair[1],syscall.SOL_SOCKET,syscall.SO_PASSCRED,1))
 own:=syscall.Ucred{Pid:int32(syscall.Getpid()),Uid:uint32(syscall.Getuid()),Gid:uint32(syscall.Getgid())};wire:=syscall.UnixCredentials(&own);n,err:=syscall.SendmsgN(pair[0],[]byte{19},wire,nil,0);must(err)
 data:=[]byte{31,37,41,43,47,53};control:=bytes.Repeat([]byte{193},80);count,oobn,flags,_,err:=syscall.Recvmsg(pair[1],data[2:4],control[4:68],syscall.MSG_CMSG_CLOEXEC);must(err);messages,err:=syscall.ParseSocketControlMessage(control[4:4+oobn]);must(err);if len(messages)!=1{panic("transport message count")};actual,err:=syscall.ParseUnixCredentials(&messages[0]);must(err)
 fmt.Println("actual transport",n,count,oobn,flags,*actual==own,data,bytes.Equal(control[:4],bytes.Repeat([]byte{193},4)),bytes.Equal(control[68:],bytes.Repeat([]byte{193},12)))
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
