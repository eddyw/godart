package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDarwinBpfBuffers(t *testing.T) {
	const fixture = `package main
import("fmt";"runtime";codec "example.com/bpfbuffers/codec")
func result(name string,err error){code:=codec.Errno(0);if err!=nil{code=err.(codec.Errno)};fmt.Println(name,int(code))}
func empty(filters []codec.BpfInsn){defer func(){v:=recover();_,typed:=v.(runtime.Error);fmt.Println("empty",v!=nil,typed,fmt.Sprint(v))}();codec.SetBpf(71,filters)}
func main(){
 n,err:=codec.BpfBuflen(71);result("buffer",err);fmt.Println("value",n)
 n,err=codec.SetBpfBuflen(71,0x1234567887654321);result("set buffer",err);fmt.Println("value",n)
 n,err=codec.BpfDatalink(71);result("datalink",err);fmt.Println("value",n)
 n,err=codec.SetBpfDatalink(71,-9223372036854775807);result("set datalink",err);fmt.Println("value",n)
 result("promisc",codec.SetBpfPromisc(71,-1));result("flush",codec.FlushBpf(71))
 name,err:=codec.BpfInterface(71,"original name");result("interface",err);fmt.Println("name",name)
 for _,name:=range []string{"","short","0123456789abcdefextra","abc\x00def","界界界界界界"}{result("set interface",codec.SetBpfInterface(71,name))}
 tv,err:=codec.BpfTimeout(71);result("timeout",err);fmt.Println("timeval",tv.Sec,tv.Usec,tv.Pad_cgo_0)
 tv=&codec.Timeval{Sec:-9223372036854775807,Usec:-2147483647,Pad_cgo_0:[4]byte{1,2,3,4}};saved:=*tv
 result("set timeout",codec.SetBpfTimeout(71,tv));fmt.Println("timeval intact",*tv==saved)
 result("nil timeout",codec.SetBpfTimeout(71,nil))
 stats,err:=codec.BpfStats(71);result("stats",err);fmt.Println("stats",stats.Recv,stats.Drop)
 result("immediate",codec.SetBpfImmediate(71,-1))
 filter:=[]codec.BpfInsn{{Code:65535,Jt:255,Jf:254,K:4294967295},{Code:32768,Jt:128,Jf:127,K:0x87654321}};before:=filter[0]
 result("filter",codec.SetBpf(71,filter));fmt.Println("filter intact",filter[0]==before);empty(nil);empty([]codec.BpfInsn{})
 result("version",codec.CheckBpfVersion(71));result("bad version",codec.CheckBpfVersion(72));result("bad major",codec.CheckBpfVersion(74))
 n,err=codec.BpfHeadercmpl(71);result("header",err);fmt.Println("value",n)
 result("set header",codec.SetBpfHeadercmpl(71,-1))
 n,err=codec.SetBpfBuflen(-1,-1);result("failed scalar",err);fmt.Println("failed value",n)
 name,err=codec.BpfInterface(-1,"original");result("failed interface",err);fmt.Println("failed name",name=="")
 tv,err=codec.BpfTimeout(-1);result("failed timeval",err);fmt.Println("failed timeval",tv==nil)
 stats,err=codec.BpfStats(-1);result("failed stats",err);fmt.Println("failed stats",stats==nil)
}
`
	const nativeMock = `package codec
import("fmt";"unsafe";"encoding/binary")
func ioctlPtr(fd int,request uint,arg unsafe.Pointer)error{
 var data,insns []byte
 if arg!=nil {
  size:=8
  switch request {
  case BIOCGETIF,BIOCSETIF:size=18
  case BIOCGRTIMEOUT,BIOCSRTIMEOUT:size=16
  case BIOCGSTATS:size=8
  case BIOCVERSION:size=4
  case BIOCSETF:
   p:=(*BpfProgram)(arg)
   insns=unsafe.Slice((*byte)(unsafe.Pointer(p.Insns)),int(p.Len)*8)
   fmt.Printf("filter program %d %x\n",p.Len,p.Pad_cgo_0)
   size=0
  }
  if size!=0 {data=unsafe.Slice((*byte)(arg),size)}
 }
 fmt.Printf("request %d %d %x %x\n",fd,request,data,insns)
 if fd==-1{return Errno(9)}
 if request==BIOCSRTIMEOUT&&arg==nil{return Errno(14)}
 switch request {
 case BIOCGBLEN,BIOCSBLEN,BIOCGDLT,BIOCGHDRCMPLT:binary.NativeEndian.PutUint32(data,0xfedcba98)
 case BIOCGETIF:copy(data,[]byte("returned device"))
 case BIOCGRTIMEOUT:copy(data,[]byte{1,0,0,0,0,0,0,128,1,0,0,128,4,3,2,1})
 case BIOCGSTATS:binary.NativeEndian.PutUint32(data,0xfedcba98);binary.NativeEndian.PutUint32(data[4:],0x87654321)
 case BIOCVERSION:major:=uint16(1);if fd==74{major=2};binary.NativeEndian.PutUint16(data,major);minor:=uint16(1);if fd==72{minor=2};binary.NativeEndian.PutUint16(data[2:],minor)
 }
 return nil
}
`
	const byteMock = `package codec
import("fmt";"encoding/binary")
func mockBpfIoctl(fd,request int,data,insns []byte)int{
 object:=data
 switch request {
 case BIOCGETIF,BIOCSETIF:
  if len(data)!=32{panic("incomplete ifreq")};for _,b:=range data[18:]{if b!=0{panic("ifreq tail")}};object=data[:18]
 case BIOCSETF:
  if len(data)!=16||binary.NativeEndian.Uint64(data[8:])!=0{panic("program pointer")}
  fmt.Printf("filter program %d %x\n",binary.NativeEndian.Uint32(data),data[4:8]);object=nil
 }
 fmt.Printf("request %d %d %x %x\n",fd,request,object,insns)
 if fd==-1{return 9}
 if request==BIOCSRTIMEOUT&&data==nil{return 14}
 switch request {
 case BIOCGBLEN,BIOCSBLEN,BIOCGDLT,BIOCGHDRCMPLT:binary.NativeEndian.PutUint32(data,0xfedcba98)
 case BIOCGETIF:copy(data,[]byte("returned device"))
 case BIOCGRTIMEOUT:copy(data,[]byte{1,0,0,0,0,0,0,128,1,0,0,128,4,3,2,1})
 case BIOCGSTATS:binary.NativeEndian.PutUint32(data,0xfedcba98);binary.NativeEndian.PutUint32(data[4:],0x87654321)
 case BIOCVERSION:major:=uint16(1);if fd==74{major=2};binary.NativeEndian.PutUint16(data,major);minor:=uint16(1);if fd==72{minor=2};binary.NativeEndian.PutUint16(data[2:],minor)
 }
 return 0
}
`
	const backendFixture = `package main
import("fmt";"runtime";"syscall/dart")
func main(){
 requests:=[]struct{request,size int}{{0x40044266,8},{0xc0044266,8},{0x4004426a,8},{0x80044278,8},{0x20004269,8},{0x20004268,0},{0x4020426b,32},{0x8020426c,32},{0x4010426e,16},{0x8010426d,16},{0x4008426f,8},{0x80044270,8},{0x80104267,16},{0x40044271,4},{0x40044274,8},{0x80044275,8}}
 for _,value:=range requests {
  data:=make([]byte,value.size);var insns []byte
  if value.request==0x80104267 {data[0]=1;insns=make([]byte,8)}
  code:=dart.SyscallBpfIoctl(-1,value.request,data,insns)
  want:=78;if runtime.GOOS=="darwin" {want=9}
  intact:=true;for i,b:=range data {expected:=byte(0);if value.request==0x80104267&&i==0{expected=1};if b!=expected{intact=false}}
  fmt.Println("backend",value.request,code==want,intact)
  if code!=want||!intact{panic("BPF backend contract")}
 }
 code:=dart.SyscallBpfIoctl(-1,0x8010426d,nil,nil);want:=78;if runtime.GOOS=="darwin"{want=14};fmt.Println("nil timeout",code==want);if code!=want{panic("BPF nil timeout")}
 code=dart.SyscallBpfIoctl(-1,0,nil,nil);fmt.Println("closed requests",code==78);if code!=78{panic("unselected ioctl")}
}
`
	for _, arch := range []string{"amd64", "arm64"} {
		t.Run(arch, func(t *testing.T) {
			nativeDir, dir := t.TempDir(), t.TempDir()
			constants, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "src/syscall/zerrors_darwin_"+arch+".go"))
			if err != nil {
				t.Fatal(err)
			}
			shared := "// Copyright 2009 The Go Authors. All rights reserved.\n// Use of this source code is governed by a BSD-style\n// license that can be found in the LICENSE file.\n\npackage codec\nimport \"syscall\"\ntype Errno=syscall.Errno\nconst EINVAL=Errno(22)\nfunc syscallErrno(code int)error{return Errno(code)}\nconst (\n"
			for _, name := range []string{"BIOCGBLEN", "BIOCSBLEN", "BIOCGDLT", "BIOCSDLT", "BIOCPROMISC", "BIOCFLUSH", "BIOCGETIF", "BIOCSETIF", "BIOCGRTIMEOUT", "BIOCSRTIMEOUT", "BIOCGSTATS", "BIOCIMMEDIATE", "BIOCSETF", "BIOCVERSION", "BIOCGHDRCMPLT", "BIOCSHDRCMPLT", "BPF_MAJOR_VERSION", "BPF_MINOR_VERSION", "IFNAMSIZ"} {
				found := false
				for _, line := range strings.Split(string(constants), "\n") {
					fields := strings.Fields(line)
					if len(fields) >= 3 && fields[0] == name && fields[1] == "=" {
						shared += line + "\n"
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("missing original constant %s", name)
				}
			}
			shared += ")\n"
			types := strings.Replace(selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/syscall/ztypes_darwin_"+arch+".go"), []string{"BpfVersion", "BpfStat", "BpfProgram", "BpfInsn", "Timeval"}), "package syscall", "package codec", 1)
			for _, d := range []string{nativeDir, dir} {
				if err := os.Mkdir(filepath.Join(d, "codec"), 0755); err != nil {
					t.Fatal(err)
				}
				writeFile(t, filepath.Join(d, "go.mod"), "module example.com/bpfbuffers\n\ngo 1.27\n")
				writeFile(t, filepath.Join(d, "main.go"), fixture)
				writeFile(t, filepath.Join(d, "codec/shared.go"), shared)
				writeFile(t, filepath.Join(d, "codec/types.go"), types)
			}
			original := strings.Replace(selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/syscall/bpf_bsd.go"), []string{"BpfBuflen", "SetBpfBuflen", "BpfDatalink", "SetBpfDatalink", "SetBpfPromisc", "FlushBpf", "ivalue", "BpfInterface", "SetBpfInterface", "BpfTimeout", "SetBpfTimeout", "BpfStats", "SetBpfImmediate", "SetBpf", "CheckBpfVersion", "BpfHeadercmpl", "SetBpfHeadercmpl"}), "package syscall", "package codec", 1)
			original = strings.Replace(original, "//go:build darwin || dragonfly || freebsd || netbsd || openbsd", "", 1)
			writeFile(t, filepath.Join(nativeDir, "codec/wrapper.go"), original)
			writeFile(t, filepath.Join(nativeDir, "codec/backend.go"), nativeMock)
			executable := filepath.Join(t.TempDir(), "native")
			run(t, nativeDir, "go", "build", "-o", executable, ".")
			want := upstreamOutcome(t, nativeDir, os.Environ(), runLimit, executable)
			if want.Exit != 0 || want.Text != "request 71 1074020966 0000000000000000 \nbuffer 0\nvalue 4275878552\nrequest 71 3221504614 2143658778563412 \nset buffer 0\nvalue 1311768469143599768\nrequest 71 1074020970 0000000000000000 \ndatalink 0\nvalue 4275878552\nrequest 71 2147762808 0100000000000080 \nset datalink 0\nvalue -9223372036854775807\nrequest 71 536887913 ffffffffffffffff \npromisc 0\nrequest 71 536887912  \nflush 0\nrequest 71 1075855979 000000000000000000000000000000000000 \ninterface 0\nname original name\nrequest 71 2149597804 000000000000000000000000000000000000 \nset interface 0\nrequest 71 2149597804 73686f727400000000000000000000000000 \nset interface 0\nrequest 71 2149597804 303132333435363738396162636465660000 \nset interface 0\nrequest 71 2149597804 616263006465660000000000000000000000 \nset interface 0\nrequest 71 2149597804 e7958ce7958ce7958ce7958ce7958ce70000 \nset interface 0\nrequest 71 1074807406 00000000000000000000000000000000 \ntimeout 0\ntimeval -9223372036854775807 -2147483647 [4 3 2 1]\nrequest 71 2148549229 01000000000000800100008001020304 \nset timeout 0\ntimeval intact true\nrequest 71 2148549229  \nnil timeout 14\nrequest 71 1074283119 0000000000000000 \nstats 0\nstats 4275878552 2271560481\nrequest 71 2147762800 ffffffffffffffff \nimmediate 0\nfilter program 2 00000000\nrequest 71 2148549223  fffffffeffffffff0080807f21436587\nfilter 0\nfilter intact true\nempty true true runtime error: index out of range [0] with length 0\nempty true true runtime error: index out of range [0] with length 0\nrequest 71 1074020977 00000000 \nversion 0\nrequest 72 1074020977 00000000 \nbad version 22\nrequest 74 1074020977 00000000 \nbad major 22\nrequest 71 1074020980 0000000000000000 \nheader 0\nvalue 4275878552\nrequest 71 2147762805 ffffffffffffffff \nset header 0\nrequest -1 3221504614 ffffffffffffffff \nfailed scalar 9\nfailed value 0\nrequest -1 1075855979 000000000000000000000000000000000000 \nfailed interface 9\nfailed name true\nrequest -1 1074807406 00000000000000000000000000000000 \nfailed timeval 9\nfailed timeval true\nrequest -1 1074283119 0000000000000000 \nfailed stats 9\nfailed stats true" {
				t.Fatalf("original BPF buffers: %+v", want)
			}
			t.Log("original SDK BPF buffers", arch, want.Text)
			data, err := os.ReadFile(filepath.Join("..", "overlay", "std", "syscall", "bpf_controls_darwin.go"))
			if err != nil {
				t.Fatal(err)
			}
			adapted := strings.Replace(string(data), "//go:build dart && darwin && (amd64 || arm64)", "", 1)
			adapted = strings.Replace(adapted, "package syscall", "package codec", 1)
			adapted = strings.Replace(adapted, "\n\t\"syscall/dart\"", "", 1)
			adapted = strings.ReplaceAll(adapted, "dart.SyscallBpfIoctl", "mockBpfIoctl")
			writeFile(t, filepath.Join(dir, "codec/wrapper.go"), adapted)
			writeFile(t, filepath.Join(dir, "codec/backend.go"), byteMock)
			t.Setenv("GOOS", "darwin")
			t.Setenv("GOARCH", arch)
			t.Setenv("CGO_ENABLED", "0")
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
						t.Fatalf("BPF buffers differ: Go=%+v Gart=%+v", want, got)
					}
				})
			}
			backendDir, backendOut := t.TempDir(), t.TempDir()
			writeFile(t, filepath.Join(backendDir, "go.mod"), "module example.com/bpfbackend\n\ngo 1.27\n")
			writeFile(t, filepath.Join(backendDir, "main.go"), backendFixture)
			if err := driver.Build(driver.Options{Dir: backendDir, Patterns: []string{"."}, Out: backendOut, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
				t.Fatal(err)
			}
			prepareDart(t, backendOut)
			for _, target := range []string{"vm", "aot", "js", "wasm"} {
				t.Run("backend/"+target, func(t *testing.T) {
					var got programResult
					if target == "vm" {
						got = upstreamOutcome(t, backendDir, os.Environ(), runLimit, "dart", filepath.Join(backendOut, "bin/main.dart"))
					} else {
						got = runUpstreamTarget(t, target, backendOut, "bin/main.dart", backendDir, []string{})
					}
					if got.Exit != 0 || strings.Count(got.Text, " true true") != 16 || !strings.HasSuffix(got.Text, "nil timeout true\nclosed requests true") {
						t.Fatalf("actual BPF backend: %+v", got)
					}
				})
			}
		})
	}
}
