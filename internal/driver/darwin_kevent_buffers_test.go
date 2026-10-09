package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestDarwinKeventBuffers(t *testing.T) {
	const fixture = `package main
import("fmt";"os";codec "example.com/keventbuffers/codec")
func main(){
 if len(os.Args)>1&&os.Args[1]=="backend"{backend();return}
 if len(os.Args)>1 {
  token:=byte(19)
  for _,value:=range []struct{name string;changes,events []codec.Kevent_t}{
   {"changes",[]codec.Kevent_t{{Ident:1,Udata:&token}},[]codec.Kevent_t{{Ident:2}}},
   {"events",[]codec.Kevent_t{{Ident:3}},[]codec.Kevent_t{{Ident:4,Udata:&token}}},
   {"tail",nil,[]codec.Kevent_t{{Ident:5},{Ident:6,Udata:&token}}},
  } {
   n,err:=codec.Kevent(-1,value.changes,value.events,nil);code,typed:=err.(codec.Errno)
   fmt.Println(value.name,n,int(code),typed,token==19)
   if n!=-1||!typed||code!=codec.ENOSYS||token!=19{panic("pointer rejection")}
  }
  return
 }
 changes:=[]codec.Kevent_t{{Ident:0xfedcba9876543210,Filter:-32768,Flags:65535,Fflags:0xfedcba98,Data:-9223372036854775807}}
 before:=changes[0]
 events:=[]codec.Kevent_t{{Ident:999,Filter:127,Flags:65535,Fflags:0xffffffff,Data:-99},{Ident:998,Filter:126,Flags:65534,Fflags:0xfffffffe,Data:-98}}
 tail:=events[1]
 timeout:=&codec.Timespec{Sec:-9223372036854775807,Nsec:9223372036854775807};saved:=*timeout
 n,err:=codec.Kevent(71,changes,events,timeout)
 v:=events[0];fmt.Println("result",n,err==nil,v.Ident,v.Filter,v.Flags,v.Fflags,v.Data,v.Udata==nil,changes[0]==before,events[1]==tail,*timeout==saved)
 n,err=codec.Kevent(-1,nil,nil,nil);code,typed:=err.(codec.Errno);fmt.Println("empty",n,int(code),typed)
 alias:=[]codec.Kevent_t{before,tail};n,err=codec.Kevent(71,alias,alias,&codec.Timespec{});fmt.Println("alias",n,err==nil,alias[0]==v,alias[1]==tail)
}
`
	const backend = `package main
import("errors";"fmt";"reflect";"syscall")
func backend(){
 events:=[]syscall.Kevent_t{{Ident:999,Filter:127,Flags:65535,Fflags:0xffffffff,Data:-99},{Ident:998,Filter:126,Flags:65534,Fflags:0xfffffffe,Data:-98}}
 before:=append([]syscall.Kevent_t(nil),events...)
 timeout:=&syscall.Timespec{};saved:=*timeout
 n,err:=syscall.Kevent(-1,nil,events,timeout);code,typed:=err.(syscall.Errno)
 fmt.Println("backend",n,int(code),typed,reflect.DeepEqual(events,before),*timeout==saved,errors.Is(err,errors.ErrUnsupported))
}
`
	const shared = `// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package codec
import "syscall"
type Errno=syscall.Errno
const ENOSYS=Errno(78)
func syscallErrno(code int)error{return Errno(code)}
`
	const nativeMock = `package codec
import("fmt";"unsafe")
func kevent(kq int,change unsafe.Pointer,nchange int,event unsafe.Pointer,nevent int,timeout *Timespec)(int,error){
 var c,e,limit []byte
 if nchange>0{c=unsafe.Slice((*byte)(change),nchange*32)}
 if nevent>0{e=unsafe.Slice((*byte)(event),nevent*32)}
 if timeout!=nil{limit=unsafe.Slice((*byte)(unsafe.Pointer(timeout)),16)}
 fmt.Printf("buffers %d %x %x %x\n",kq,c,e,limit)
 if kq==-1{return -1,Errno(9)}
 output:=Kevent_t{Ident:0x8877665544332211,Filter:-32768,Flags:65535,Fflags:0xabcdef01,Data:-9223372036854775807}
 copy(e,unsafe.Slice((*byte)(unsafe.Pointer(&output)),32))
 return 1,nil
}
`
	const byteMock = `package codec
import("fmt";"encoding/hex")
func mockKevent(kq int,changes,events,timeout []byte)(int,int){
 fmt.Printf("buffers %d %x %x %x\n",kq,changes,events,timeout)
 if kq==-1{return -1,9}
 output,err:=hex.DecodeString("11223344556677880080ffff01efcdab01000000000000800000000000000000");if err!=nil{panic(err)}
 copy(events,output);return 1,0
}
`
	for _, arch := range []string{"amd64", "arm64"} {
		t.Run(arch, func(t *testing.T) {
			nativeDir, dir := t.TempDir(), t.TempDir()
			for _, d := range []string{nativeDir, dir} {
				if err := os.Mkdir(filepath.Join(d, "codec"), 0755); err != nil {
					t.Fatal(err)
				}
				writeFile(t, filepath.Join(d, "go.mod"), "module example.com/keventbuffers\n\ngo 1.27\n")
				writeFile(t, filepath.Join(d, "main.go"), fixture)
				writeFile(t, filepath.Join(d, "codec/shared.go"), shared)
				writeFile(t, filepath.Join(d, "codec/types.go"), strings.Replace(selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/syscall/ztypes_darwin_"+arch+".go"), []string{"Kevent_t", "Timespec"}), "package syscall", "package codec", 1))
			}
			original := selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/syscall/syscall_bsd.go"), []string{"Kevent"})
			original = strings.Replace(original, "package syscall", "package codec", 1)
			writeFile(t, filepath.Join(nativeDir, "backend.go"), "package main\nfunc backend(){}\n")
			writeFile(t, filepath.Join(dir, "backend.go"), backend)
			original = strings.Replace(original, "//go:build darwin || dragonfly || freebsd || netbsd || openbsd", "", 1)
			writeFile(t, filepath.Join(nativeDir, "codec/wrapper.go"), original)
			writeFile(t, filepath.Join(nativeDir, "codec/backend.go"), nativeMock)
			native := filepath.Join(t.TempDir(), "native")
			run(t, nativeDir, "go", "build", "-o", native, ".")
			want := upstreamOutcome(t, nativeDir, os.Environ(), runLimit, native)
			if want.Exit != 0 || !strings.Contains(want.Text, "result 1 true 9833440827789222417 -32768 65535 2882400001 -9223372036854775807 true true true true") || !strings.Contains(want.Text, "empty -1 9 true") || !strings.HasSuffix(want.Text, "alias 1 true true true") {
				t.Fatalf("original Kevent bytes: %+v", want)
			}
			t.Log("original SDK Kevent buffers", arch, want.Text)
			data, err := os.ReadFile(filepath.Join("..", "overlay", "std", "syscall", "kevent_darwin.go"))
			if err != nil {
				t.Fatal(err)
			}
			adapted := strings.Replace(string(data), "//go:build dart && darwin && (amd64 || arm64)", "", 1)
			adapted = strings.Replace(adapted, "package syscall", "package codec", 1)
			adapted = strings.Replace(adapted, "\n\t\"syscall/dart\"", "", 1)
			adapted = strings.Replace(adapted, "dart.SyscallKevent", "mockKevent", 1)
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
					var command string
					var commandArgs []string
					if target != "vm" {
						runner := "import 'bin/main.dart' as tests;\nvoid main(List<String> args) { tests.main(args); }\n"
						if target == "js" || target == "wasm" {
							runner = `import 'dart:js_interop';
import 'bin/main.dart' as tests;
@JS('godartKeventArguments')
external JSArray<JSString> get arguments;
void main() {
  tests.main(arguments.toDart.map((argument) => argument.toDart).toList());
}
`
						}
						writeFile(t, filepath.Join(out, "runner.dart"), runner)
						var compiled programResult
						switch target {
						case "aot":
							command = filepath.Join(out, "tests.exe")
							compiled = upstreamOutcome(t, out, nil, runLimit, "dart", "compile", "exe", "runner.dart", "-o", command)
						case "js":
							compiled = upstreamOutcome(t, out, nil, runLimit, "dart", "compile", "js", "runner.dart", "-o", "main.js")
							writeFile(t, filepath.Join(out, "run.js"), "globalThis.self = globalThis;\nglobalThis.godartKeventArguments = process.argv.slice(2);\nrequire('./main.js');\n")
							command = "node"
							commandArgs = []string{filepath.Join(out, "run.js")}
						case "wasm":
							compiled = upstreamOutcome(t, out, nil, 16*time.Minute, "dart", "compile", "wasm", "runner.dart", "-o", "main.wasm")
							writeFile(t, filepath.Join(out, "run.mjs"), `import {compile} from './main.mjs';
import {readFile} from 'node:fs/promises';
globalThis.godartKeventArguments = process.argv.slice(2);
const app = await compile(await readFile(new URL('./main.wasm', import.meta.url)));
const instance = await app.instantiate({});
instance.invokeMain();
`)
							command = "node"
							commandArgs = []string{filepath.Join(out, "run.mjs")}
						}
						if compiled.Exit != 0 {
							t.Fatalf("Kevent compile %s: %+v", target, compiled)
						}
					}
					for _, mode := range []string{"bytes", "unsupported", "backend"} {
						t.Run(mode, func(t *testing.T) {
							args := []string{}
							expected := want
							if mode == "unsupported" {
								args = []string{mode}
								expected = programResult{Exit: 0, Text: "changes -1 78 true true\nevents -1 78 true true\ntail -1 78 true true"}
							}
							if mode == "backend" {
								args = []string{mode}
								expected = programResult{Exit: 0, Text: "backend -1 78 true true true true"}
								if runtime.GOOS == "darwin" && (target == "vm" || target == "aot") {
									expected.Text = "backend -1 9 true true true false"
								}
							}
							var got programResult
							if target == "vm" {
								got = upstreamOutcome(t, dir, os.Environ(), runLimit, "dart", append([]string{filepath.Join(out, "bin/main.dart")}, args...)...)
							} else {
								got = upstreamOutcome(t, dir, nil, upstreamLimit(), command, append(append([]string{}, commandArgs...), args...)...)
							}
							if got != expected {
								t.Fatalf("Kevent %s differs: Go=%+v Gart=%+v", mode, expected, got)
							}
						})
					}
				})
			}
		})
	}
}
