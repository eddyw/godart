package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func routingDeclarations(t *testing.T, name string) map[string]string {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	set := token.NewFileSet()
	file, err := parser.ParseFile(set, name, data, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	result := map[string]string{}
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			start := d.Pos()
			if d.Doc != nil {
				start = d.Doc.Pos()
			}
			result[d.Name.Name] = string(data[set.Position(start).Offset:set.Position(d.End()).Offset])
			if d.Recv != nil {
				if ptr, ok := d.Recv.List[0].Type.(*ast.StarExpr); ok {
					if receiver, ok := ptr.X.(*ast.Ident); ok {
						result[receiver.Name+"."+d.Name.Name] = result[d.Name.Name]
					}
				}
			}

		case *ast.GenDecl:
			for _, spec := range d.Specs {
				if typ, ok := spec.(*ast.TypeSpec); ok {
					result[typ.Name.Name] = "type " + string(data[set.Position(typ.Pos()).Offset:set.Position(typ.End()).Offset])
				}
			}
		}
	}
	return result
}
func routingReference(t *testing.T, arch, source string) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/routingreference\n\ngo 1.27\n")
	sdk := filepath.Join(runtime.GOROOT(), "src", "syscall")
	ref := filepath.Join(dir, "routeorig")
	if err := os.MkdirAll(ref, 0755); err != nil {
		t.Fatal(err)
	}
	rewrite := func(s string) string {
		return strings.Replace(regexp.MustCompile(`(?m)^//go:build.*\n`).ReplaceAllString(s, ""), "package syscall", "package routeorig", 1)
	}
	for _, kind := range []string{"ztypes", "zerrors"} {
		data, err := os.ReadFile(filepath.Join(sdk, kind+"_darwin_"+arch+".go"))
		if err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(ref, kind+".go"), rewrite(string(data)))
	}
	data, err := os.ReadFile(filepath.Join(sdk, "route_bsd.go"))
	if err != nil {
		t.Fatal(err)
	}
	decls := routingDeclarations(t, filepath.Join(sdk, "route_bsd.go"))
	writeFile(t, filepath.Join(ref, "route_bsd.go"), rewrite(strings.Replace(string(data), decls["RouteRIB"], "", 1)))
	data, err = os.ReadFile(filepath.Join(sdk, "route_darwin.go"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(ref, "route_selected.go"), rewrite(string(data)))
	darwin := routingDeclarations(t, filepath.Join(sdk, "syscall_darwin.go"))
	bsd := routingDeclarations(t, filepath.Join(sdk, "syscall_bsd.go"))
	bridge := `// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package routeorig
import("syscall";"unsafe")
const darwin64Bit=true
const netbsd32Bit=false
type Errno=syscall.Errno
type Signal=syscall.Signal
type Sockaddr interface{}
type SockaddrInet4=syscall.SockaddrInet4
type SockaddrInet6=syscall.SockaddrInet6
type SockaddrUnix=syscall.SockaddrUnix
` + darwin["SockaddrDatalink"] + "\n" + bsd["anyToSockaddr"] + "\n"
	writeFile(t, filepath.Join(ref, "bridge.go"), bridge)
	writeFile(t, filepath.Join(dir, "main.go"), strings.Replace(source, `"syscall"`, `syscall "example.com/routingreference/routeorig"`, 1))
	return dir
}
func TestDarwinRoutingByteParsers(t *testing.T) {
	source := `package main
import("bytes";"encoding/binary";"errors";"fmt";"os";"syscall")
func must(err error){if err!=nil{panic(err)}}
func messages(){
 for _,data:=range [][]byte{nil,{}, {1},{1,2},{1,2,3}}{msgs,err:=syscall.ParseRoutingMessage(data);fmt.Println("short",len(data),msgs==nil,err==nil)}
 for _,typ:=range []byte{syscall.RTM_ADD,syscall.RTM_DELETE,syscall.RTM_CHANGE,syscall.RTM_GET,syscall.RTM_LOSING,syscall.RTM_REDIRECT,syscall.RTM_MISS,syscall.RTM_LOCK,syscall.RTM_RESOLVE,syscall.RTM_IFINFO,syscall.RTM_NEWADDR,syscall.RTM_DELADDR,syscall.RTM_NEWMADDR2,syscall.RTM_DELMADDR}{
  size:=syscall.SizeofRtMsghdr
  switch typ{case syscall.RTM_IFINFO:size=syscall.SizeofIfMsghdr;case syscall.RTM_NEWADDR,syscall.RTM_DELADDR:size=syscall.SizeofIfaMsghdr;case syscall.RTM_NEWMADDR2,syscall.RTM_DELMADDR:size=syscall.SizeofIfmaMsghdr2}
  data:=make([]byte,size+4);for i:=range data{data[i]=byte(i*17+3)};binary.LittleEndian.PutUint16(data,uint16(len(data)));data[2]=syscall.RTM_VERSION;data[3]=typ
  msgs,err:=syscall.ParseRoutingMessage(data);must(err);if len(msgs)!=1{panic("message count")};var header any;var payload []byte
  switch msg:=msgs[0].(type){case *syscall.RouteMessage:header=msg.Header;payload=msg.Data;case *syscall.InterfaceMessage:header=msg.Header;payload=msg.Data;case *syscall.InterfaceAddrMessage:header=msg.Header;payload=msg.Data;case *syscall.InterfaceMulticastAddrMessage:header=msg.Header;payload=msg.Data;default:panic("message type")}
  var encoded bytes.Buffer;must(binary.Write(&encoded,binary.LittleEndian,header));fmt.Println("message",typ,len(encoded.Bytes())==size&&bytes.Equal(encoded.Bytes(),data[:size]),bytes.Equal(payload,data[size:]),len(payload),cap(payload));data[size]^=255;fmt.Println("alias",payload[0]==data[size]);before:=encoded.Bytes()[4];data[4]^=255;var retained bytes.Buffer;must(binary.Write(&retained,binary.LittleEndian,header));fmt.Println("header copy",retained.Bytes()[4]==before)
 }
 for _,data:=range [][]byte{{4,0,syscall.RTM_VERSION,255},{4,0,0,255}}{msgs,err:=syscall.ParseRoutingMessage(data);code,typed:=err.(syscall.Errno);fmt.Println("unsupported",msgs==nil,int(code),typed)}
 for _,data:=range [][]byte{{5,0,syscall.RTM_VERSION,255},{4,0,syscall.RTM_VERSION,syscall.RTM_ADD}}{func(){defer func(){fmt.Println("invalid panic",recover()!=nil)}();_,err:=syscall.ParseRoutingMessage(data);fmt.Println("unexpected result",err)}()}
}

func parse(label string,msg syscall.RoutingMessage){addresses,err:=syscall.ParseRoutingSockaddr(msg);code,typed:=err.(syscall.Errno);fmt.Println(label,len(addresses),addresses==nil,int(code),typed);for i,address:=range addresses{if address==nil{continue};switch a:=address.(type){case *syscall.SockaddrInet4:fmt.Println("inet4",i,a.Port,a.Addr);case *syscall.SockaddrInet6:fmt.Println("inet6",i,a.Port,a.ZoneId,a.Addr);case *syscall.SockaddrDatalink:fmt.Println("link",i,a.Len,a.Family,a.Index,a.Type,a.Nlen,a.Alen,a.Slen,len(a.Data),a.Data[:8]);default:panic("address type")}}}
func addresses(){v4:=[]byte{16,syscall.AF_INET,0x12,0x34,1,2,3,4,0,0,0,0,0,0,0,0};v6:=make([]byte,28);v6[0]=28;v6[1]=syscall.AF_INET6;v6[2]=0xab;v6[3]=0xcd;for i:=8;i<24;i++{v6[i]=byte(i*17+3)};v6[24]=0x78;v6[25]=0x56;v6[26]=0x34;v6[27]=0x12;link:=[]byte{16,syscall.AF_LINK,0x34,0x12,7,3,4,1,65,66,67,128,255,1,2,3}
parse("route empty",&syscall.RouteMessage{});parse("interface empty",&syscall.InterfaceMessage{});parse("address empty",&syscall.InterfaceAddrMessage{});parse("multicast empty",&syscall.InterfaceMulticastAddrMessage{})
parse("route inet4",&syscall.RouteMessage{Header:syscall.RtMsghdr{Addrs:syscall.RTA_DST},Data:v4});parse("route inet6",&syscall.RouteMessage{Header:syscall.RtMsghdr{Addrs:syscall.RTA_DST},Data:v6});parse("route link",&syscall.RouteMessage{Header:syscall.RtMsghdr{Addrs:syscall.RTA_DST},Data:link});parse("interface link",&syscall.InterfaceMessage{Header:syscall.IfMsghdr{Addrs:syscall.RTA_IFP},Data:link});parse("address inet4",&syscall.InterfaceAddrMessage{Header:syscall.IfaMsghdr{Addrs:syscall.RTA_IFA},Data:v4});parse("multicast inet6",&syscall.InterfaceMulticastAddrMessage{Header:syscall.IfmaMsghdr2{Addrs:syscall.RTA_IFA},Data:v6})
mask:=[]byte{8,0,255,255,255,128,0,0};parse("route v4 mask",&syscall.RouteMessage{Header:syscall.RtMsghdr{Addrs:syscall.RTA_DST|syscall.RTA_NETMASK},Data:append(append([]byte{},v4...),mask...)});parse("route v6 mask",&syscall.RouteMessage{Header:syscall.RtMsghdr{Addrs:syscall.RTA_DST|syscall.RTA_NETMASK},Data:append(append([]byte{},v6...),mask...)})
for _,data:=range [][]byte{{1,2,3},{16,syscall.AF_INET,0,0},{28,syscall.AF_INET6,0,0},{8,syscall.AF_LINK,0,0,7,255,0,0}}{parse("short",&syscall.RouteMessage{Header:syscall.RtMsghdr{Addrs:syscall.RTA_DST},Data:data})}
for _,msg:=range []syscall.RoutingMessage{nil,(*syscall.RouteMessage)(nil),(*syscall.InterfaceMessage)(nil),(*syscall.InterfaceAddrMessage)(nil),(*syscall.InterfaceMulticastAddrMessage)(nil)}{func(){defer func(){fmt.Println("nil panic",recover()!=nil)}();_,err:=syscall.ParseRoutingSockaddr(msg);fmt.Println("unexpected result",err)}()}
}

func main(){
 if len(os.Args)>1 {
  for _,data:=range [][]byte{{0,0,syscall.RTM_VERSION,255},{0,0,0,255}}{
   msgs,err:=syscall.ParseRoutingMessage(data);code,typed:=err.(syscall.Errno);fmt.Println("zero frame",msgs==nil,int(code),typed,errors.Is(err,errors.ErrUnsupported))
  }
  return
 }
 messages();addresses()
}
`
	want := programResult{Exit: 0, Text: "short 0 true true\nshort 0 true true\nshort 1 true true\nshort 2 true true\nshort 3 true true\nmessage 1 true true 4 4\nalias true\nheader copy true\nmessage 2 true true 4 4\nalias true\nheader copy true\nmessage 3 true true 4 4\nalias true\nheader copy true\nmessage 4 true true 4 4\nalias true\nheader copy true\nmessage 5 true true 4 4\nalias true\nheader copy true\nmessage 6 true true 4 4\nalias true\nheader copy true\nmessage 7 true true 4 4\nalias true\nheader copy true\nmessage 8 true true 4 4\nalias true\nheader copy true\nmessage 11 true true 4 4\nalias true\nheader copy true\nmessage 14 true true 4 4\nalias true\nheader copy true\nmessage 12 true true 4 4\nalias true\nheader copy true\nmessage 13 true true 4 4\nalias true\nheader copy true\nmessage 19 true true 4 4\nalias true\nheader copy true\nmessage 16 true true 4 4\nalias true\nheader copy true\nunsupported true 0 false\nunsupported true 22 true\ninvalid panic true\ninvalid panic true\nroute empty 8 false 0 false\ninterface empty 0 true 0 false\naddress empty 8 false 0 false\nmulticast empty 8 false 0 false\nroute inet4 8 false 0 false\ninet4 0 4660 [1 2 3 4]\nroute inet6 8 false 0 false\ninet6 0 43981 305419896 [139 156 173 190 207 224 241 2 19 36 53 70 87 104 121 138]\nroute link 8 false 0 false\nlink 0 16 18 4660 7 3 4 1 12 [65 66 67 -128 -1 1 2 3]\ninterface link 8 false 0 false\nlink 4 16 18 4660 7 3 4 1 12 [65 66 67 -128 -1 1 2 3]\naddress inet4 8 false 0 false\ninet4 5 4660 [1 2 3 4]\nmulticast inet6 8 false 0 false\ninet6 5 43981 305419896 [139 156 173 190 207 224 241 2 19 36 53 70 87 104 121 138]\nroute v4 mask 8 false 0 false\ninet4 0 4660 [1 2 3 4]\ninet4 2 0 [255 128 0 0]\nroute v6 mask 8 false 0 false\ninet6 0 43981 305419896 [139 156 173 190 207 224 241 2 19 36 53 70 87 104 121 138]\ninet6 2 0 0 [0 255 255 255 128 0 0 0 0 0 0 0 0 0 0 0]\nshort 8 false 0 false\nshort 0 true 22 true\nshort 0 true 22 true\nshort 0 true 22 true\nnil panic true\nnil panic true\nnil panic true\nnil panic true\nnil panic true"}
	for _, arch := range []string{"amd64", "arm64"} {
		ref := routingReference(t, arch, source)
		native := filepath.Join(t.TempDir(), "native")
		run(t, ref, "go", "build", "-o", native, ".")
		got := upstreamOutcome(t, ref, os.Environ(), runLimit, native)
		if got != want {
			t.Fatalf("original selected Darwin routing %s: %+v", arch, got)
		}
		t.Log("original SDK routing reference", arch)
	}
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/routingbytes\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), source)
	t.Setenv("GOOS", "darwin")
	for _, arch := range []string{"amd64", "arm64"} {
		t.Run(arch, func(t *testing.T) {
			t.Setenv("GOARCH", arch)
			out := t.TempDir()
			if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
				t.Fatal(err)
			}
			prepareDart(t, out)
			for _, target := range []string{"vm", "aot", "js", "wasm"} {
				t.Run(target, func(t *testing.T) {
					var command string
					var commandArgs []string
					if target != "vm" {
						command, commandArgs = compileFixtureTarget(t, target, out)
					}
					for _, mode := range []string{"normal", "unsupported"} {
						t.Run(mode, func(t *testing.T) {
							args := []string{}
							expected := want
							if mode == "unsupported" {
								args = append(args, mode)
								expected = programResult{Exit: 0, Text: "zero frame true 78 true true\nzero frame true 78 true true"}
							}
							var got programResult
							if target == "vm" {
								got = upstreamOutcome(t, dir, os.Environ(), runLimit, "dart", append([]string{filepath.Join(out, "bin/main.dart")}, args...)...)
							} else {
								got = upstreamOutcome(t, dir, nil, upstreamLimit(), command, append(append([]string{}, commandArgs...), args...)...)
							}
							if got != expected {
								t.Fatalf("routing bytes %s differ: want=%+v Gart=%+v", mode, expected, got)
							}
						})
					}
				})
			}
		})
	}
}
