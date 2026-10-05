package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDarwinRoutingByteBoundaries(t *testing.T) {
	source := `package main
import("fmt";"encoding/binary";"syscall"; codec "example.com/routingboundary/codec")
func frame(kind byte, size, spare int) []byte {
 b:=make([]byte,size,size+spare);for i:=range b {b[i]=byte(i*37+11)}
 binary.NativeEndian.PutUint16(b,uint16(size));b[2]=syscall.RTM_VERSION;b[3]=kind;return b
}
func joined(a,b []byte) []byte {data:=make([]byte,len(a)+len(b),len(a)+len(b)+19);copy(data,a);copy(data[len(a):],b);return data}
func inspect(label string,b []byte) {
 defer func(){if recover()!=nil {fmt.Println(label,"panic")}}()
 msgs,err:=syscall.ParseRoutingMessage(b);fmt.Println(label,len(msgs),msgs==nil,err)
 for _,msg:=range msgs {var data []byte;switch m:=msg.(type){case *syscall.RouteMessage:data=m.Data;case *syscall.InterfaceMessage:data=m.Data;case *syscall.InterfaceAddrMessage:data=m.Data;case *syscall.InterfaceMulticastAddrMessage:data=m.Data};fmt.Println("payload",len(data),cap(data));if len(data)>0 {data[0]^=255;fmt.Println("alias",b[len(b)-len(data)])}}
}
func main(){
 for _,v:=range []struct{kind byte;size int}{{syscall.RTM_GET,syscall.SizeofRtMsghdr},{syscall.RTM_IFINFO,syscall.SizeofIfMsghdr},{syscall.RTM_NEWADDR,syscall.SizeofIfaMsghdr},{syscall.RTM_NEWMADDR2,syscall.SizeofIfmaMsghdr2}} {
  inspect("spare",frame(v.kind,v.size+3,19));inspect("empty",frame(v.kind,v.size,19));inspect("short",frame(v.kind,4,0))
 }
 b:=frame(syscall.RTM_GET,syscall.SizeofRtMsghdr+3,0);b=joined(b,frame(syscall.RTM_NEWADDR,syscall.SizeofIfaMsghdr+2,0));inspect("concat",b)
 b=frame(255,4,0);b=joined(b,frame(syscall.RTM_GET,syscall.SizeofRtMsghdr,0));inspect("skip",b)
 b=frame(255,4,0);b[2]=0;b=joined(b,frame(syscall.RTM_GET,syscall.SizeofRtMsghdr,0));inspect("version",b)
 for _,family:=range []byte{syscall.AF_INET,syscall.AF_INET6} {for _,length:=range []byte{0,1,4,5,8,16,28} {
  first:=make([]byte,28);first[0]=28;first[1]=family;if family==syscall.AF_INET {first=first[:16];first[0]=16};mask:=make([]byte,28);for i:=range mask{mask[i]=byte(128+i)};mask[0]=length;mask[1]=0
  msg:=&syscall.RouteMessage{Header:syscall.RtMsghdr{Addrs:syscall.RTA_DST|syscall.RTA_NETMASK},Data:append(first,mask...)};sas,err:=syscall.ParseRoutingSockaddr(msg);fmt.Println("mask",family,length,err);if err==nil {switch sa:=sas[syscall.RTAX_NETMASK].(type){case *syscall.SockaddrInet4:fmt.Println(sa.Addr);case *syscall.SockaddrInet6:fmt.Println(sa.Addr)}}
 }}
 codec.Inspect()
}
`
	common := `
func Inspect(){
 sa:=&SockaddrDatalink{Len:255,Family:255,Index:65535,Type:255,Nlen:255,Alen:128,Slen:127,Data:[12]int8{-128,-1,0,1,2,3,4,5,6,7,126,127}}
 for i:=0;i<3;i++ {
  b,err:=marshal(sa);fmt.Println("datalink",b,err,sa.raw)
  if i==0 {sa.Len=0;sa.Family=0;sa.Index=1;sa.Type=0;sa.Nlen=0;sa.Alen=0;sa.Slen=0;sa.Data=[12]int8{}} else {sa.Index=0;sa.Data[0]=99}
 }
 func(){defer func(){fmt.Println("datalink nil",recover()!=nil)}();marshal(nil)}()
}
`
	for _, arch := range []string{"amd64", "arm64"} {
		t.Run(arch, func(t *testing.T) {
			ref := routingReference(t, arch, strings.Replace(source, `codec "example.com/routingboundary/codec"`, `codec "example.com/routingreference/routeorig"`, 1))
			bsd := routingDeclarations(t, filepath.Join(runtime.GOROOT(), "src/syscall/syscall_bsd.go"))
			original := `// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package routeorig
import("fmt";"unsafe")
` + bsd["SockaddrDatalink.sockaddr"] + `
func marshal(sa *SockaddrDatalink)([]byte,error){p,n,err:=sa.sockaddr();if err!=nil{return nil,err};return append([]byte(nil),(*[SizeofSockaddrDatalink]byte)(p)[:n]...),nil}
` + common
			writeFile(t, filepath.Join(ref, "routeorig/datalink_reference.go"), original)
			native := filepath.Join(t.TempDir(), "native")
			run(t, ref, "go", "build", "-o", native, ".")
			want := upstreamOutcome(t, ref, os.Environ(), runLimit, native)
			if want.Exit != 0 {
				t.Fatal(want)
			}
			t.Log("original SDK routing boundaries", arch, want.Text)
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/routingboundary\n\ngo 1.27\n")
			writeFile(t, filepath.Join(dir, "main.go"), source)
			if err := os.MkdirAll(filepath.Join(dir, "codec"), 0755); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(filepath.Join(filepath.Join(runtimeDir(t), "../.."), "internal/overlay/std/syscall/socket_datalink_darwin.go"))
			if err != nil {
				t.Fatal(err)
			}
			adapter := strings.Replace(strings.Replace(string(data), "//go:build dart && darwin && (amd64 || arm64)", "", 1), "package syscall", "package codec", 1)
			writeFile(t, filepath.Join(dir, "codec/adapter.go"), adapter)
			writeFile(t, filepath.Join(dir, "codec/inspect.go"), `package codec
import("fmt";"syscall")
type RawSockaddrDatalink=syscall.RawSockaddrDatalink
const AF_LINK=syscall.AF_LINK
const SizeofSockaddrDatalink=syscall.SizeofSockaddrDatalink
var EINVAL=syscall.EINVAL
func marshal(sa *SockaddrDatalink)([]byte,error){return sa.sockaddrBytes()}
`+common)
			t.Setenv("GOOS", "darwin")
			t.Setenv("GOARCH", arch)
			out := t.TempDir()
			if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
				t.Fatal(err)
			}
			prepareDart(t, out)
			for _, target := range []string{"vm", "aot", "js", "wasm"} {
				t.Run(target, func(t *testing.T) {
					var got programResult
					if target == "vm" {
						got = upstreamOutcome(t, dir, os.Environ(), runLimit, "dart", filepath.Join(out, "bin/main.dart"))
					} else {
						got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
					}
					if got != want {
						t.Fatalf("routing boundaries differ: want=%+v Gart=%+v", want, got)
					}
				})
			}
		})
	}
}
