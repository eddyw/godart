package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLinuxNetlinkKernel(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Netlink requires Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/accept\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
 "encoding/binary"
 "fmt"
 "syscall"
)

func must(err error) { if err != nil { panic(err) } }
func socket() int { fd,err:=syscall.Socket(syscall.AF_NETLINK,syscall.SOCK_RAW|syscall.SOCK_CLOEXEC,syscall.NETLINK_ROUTE);must(err);return fd }
func name(fd int) *syscall.SockaddrNetlink { sa,err:=syscall.Getsockname(fd);must(err);return sa.(*syscall.SockaddrNetlink) }
func main() {
 a,b:=socket(),socket();defer syscall.Close(a);defer syscall.Close(b)
 input:=&syscall.SockaddrNetlink{Family:syscall.AF_INET,Pad:123}
 must(syscall.Bind(a,input));must(syscall.Bind(b,&syscall.SockaddrNetlink{}))
 first,second:=name(a),name(a);other:=name(b)
 fmt.Println("names",first.Family==syscall.AF_NETLINK,first.Pid!=0,other.Pid!=0,first.Pid!=other.Pid,first==second,first.Pid==second.Pid,first.Pad,first.Groups,input.Family,input.Pad)
 first.Pid=0;first.Groups=1
 fmt.Println("independent",name(a).Pid==second.Pid,name(a).Groups==second.Groups)
 duplicate,err:=syscall.Dup(a);must(err);defer syscall.Close(duplicate)
 fmt.Println("duplicate",name(duplicate).Pid==second.Pid)
 must(syscall.Connect(a,&syscall.SockaddrNetlink{Family:syscall.AF_INET}))
 peer,err:=syscall.Getpeername(a);must(err);kernel:=peer.(*syscall.SockaddrNetlink)
 fmt.Println("peer",kernel.Family==syscall.AF_NETLINK,kernel.Pid,kernel.Groups)
 request:=make([]byte,syscall.NLMSG_HDRLEN+syscall.SizeofRtGenmsg)
 binary.NativeEndian.PutUint32(request,uint32(len(request)));binary.NativeEndian.PutUint16(request[4:],syscall.RTM_GETLINK)
 binary.NativeEndian.PutUint16(request[6:],syscall.NLM_F_DUMP|syscall.NLM_F_REQUEST);binary.NativeEndian.PutUint32(request[8:],37);request[16]=syscall.AF_UNSPEC
 must(syscall.Sendto(duplicate,request,0,&syscall.SockaddrNetlink{}))
 buffer:=make([]byte,syscall.Getpagesize());done,records,attributes,loopback:=false,0,0,false
 for !done {
  n,source,err:=syscall.Recvfrom(a,buffer,0);must(err)
  actual:=source.(*syscall.SockaddrNetlink);if actual.Family!=syscall.AF_NETLINK || actual.Pid!=0 {panic("kernel source")}
  messages,err:=syscall.ParseNetlinkMessage(buffer[:n]);must(err)
  for _,m:=range messages {
   if m.Header.Seq!=37 || m.Header.Pid!=second.Pid {panic("kernel recipient")}
   if m.Header.Type==syscall.NLMSG_DONE {done=true;continue}
   if m.Header.Type!=syscall.RTM_NEWLINK {panic("link type")};records++
   attrs,err:=syscall.ParseNetlinkRouteAttr(&m);must(err);attributes+=len(attrs)
   for _,attr:=range attrs {if int(attr.Attr.Len)!=syscall.SizeofRtAttr+len(attr.Value){panic("attribute length")};if attr.Attr.Type==syscall.IFLA_IFNAME && string(attr.Value)=="lo\x00" {loopback=true}}
  }
 }
 fmt.Println("raw dump",done,records>0,attributes>0,loopback)
 for _,query:=range []struct{proto,kind,prefix int}{{syscall.RTM_GETLINK,syscall.RTM_NEWLINK,syscall.SizeofIfInfomsg},{syscall.RTM_GETADDR,syscall.RTM_NEWADDR,syscall.SizeofIfAddrmsg},{syscall.RTM_GETROUTE,syscall.RTM_NEWROUTE,syscall.SizeofRtMsg}} {
  data,err:=syscall.NetlinkRIB(query.proto,syscall.AF_UNSPEC);must(err)
  saved:=append([]byte(nil),data...)
  messages,err:=syscall.ParseNetlinkMessage(data);must(err)
  port:=messages[0].Header.Pid;done,records,attributes=false,0,0
  for _,m:=range messages {
   if m.Header.Seq!=1 || m.Header.Pid!=port || port==0 {panic("RIB recipient")}
   if m.Header.Type==syscall.NLMSG_DONE {done=true;continue}
   if int(m.Header.Type)!=query.kind || len(m.Data)<query.prefix {panic("RIB record")};records++
   attrs,err:=syscall.ParseNetlinkRouteAttr(&m);must(err);attributes+=len(attrs)
   for _,attr:=range attrs {if int(attr.Attr.Len)!=syscall.SizeofRtAttr+len(attr.Value){panic("RIB attribute")}}
  }
  fmt.Println("RIB",query.proto,done,records>0,attributes>0)
  _,err=syscall.NetlinkRIB(query.proto,syscall.AF_UNSPEC);must(err)
  fmt.Println("retained",string(data)==string(saved))
 }
 fd:=socket();must(syscall.Close(fd));sa,err:=syscall.Getsockname(fd)
 fmt.Println("closed",sa==nil,err==syscall.EBADF)
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
