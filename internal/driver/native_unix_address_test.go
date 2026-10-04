package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNativeUnixAddressCalls(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("Native Unix address calls require Linux or macOS")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/unixaddress\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
 "fmt"
 "os"
 "strings"
 "syscall"
)

func errline(label string,err error) {
 code,typed:=err.(syscall.Errno)
 fmt.Println(label,code,typed)
}

func socket() int {
 fd,err:=syscall.Socket(syscall.AF_UNIX,syscall.SOCK_DGRAM,0)
 if err!=nil { panic(err) }
 if err:=syscall.SetNonblock(fd,true);err!=nil { panic(err) }
 return fd
}

func read(label string,fd int) {
 data:=make([]byte,16)
 n,err:=syscall.Read(fd,data)
 fmt.Println(label,n)
 errline("read",err)
 if n>0 { fmt.Println("actual bytes",data[:n]) }
}

func main() {
 dir,err:=os.MkdirTemp("","raw-unix-address")
 if err!=nil { panic(err) }
 defer os.RemoveAll(dir)
 if err:=os.Chdir(dir);err!=nil { panic(err) }
 server,client:=socket(),socket()
 destination:=&syscall.SockaddrUnix{Name:"server"}
 source:=&syscall.SockaddrUnix{Name:"client"}
 errline("bind server",syscall.Bind(server,destination))
 errline("bind client",syscall.Bind(client,source))
 errline("repeat bind",syscall.Bind(client,source))
 duplicate,err:=syscall.Dup(client)
 if err!=nil { panic(err) }
 errline("connect duplicate",syscall.Connect(duplicate,destination))
 n,err:=syscall.Write(client,[]byte{0,128,255})
 fmt.Println("connected write",n)
 errline("write",err)
 read("server packet",server)
 errline("connected send",syscall.Sendto(client,[]byte{42,0},0,nil))
 read("server next packet",server)
 errline("response",syscall.Sendto(server,[]byte{255,0,128},0,source))
 read("client packet",client)
 errline("duplicate close",syscall.Close(duplicate))
 value,err:=syscall.GetsockoptInt(client,syscall.SOL_SOCKET,syscall.SO_TYPE)
 fmt.Println("client owned",value==syscall.SOCK_DGRAM)
 errline("option",err)
 errline("missing peer",syscall.Connect(client,&syscall.SockaddrUnix{Name:"missing"}))
 for _,call:=range []func()error{
  func()error{return syscall.Bind(-1,&syscall.SockaddrUnix{Name:strings.Repeat("x",109)})},
  func()error{return syscall.Connect(-1,&syscall.SockaddrUnix{Name:strings.Repeat("x",109)})},
  func()error{return syscall.Sendto(-1,nil,0,&syscall.SockaddrUnix{Name:strings.Repeat("x",109)})},
  func()error{return syscall.Bind(-1,&syscall.SockaddrUnix{Name:"valid"})},
  func()error{return syscall.Connect(-1,&syscall.SockaddrUnix{Name:"valid"})},
 } { errline("invalid",call()) }
 unnamed:=socket()
 errline("empty bind",syscall.Bind(unnamed,&syscall.SockaddrUnix{}))
 if err:=syscall.Close(unnamed);err!=nil { panic(err) }
 first,second:=socket(),socket()
 cached:=&syscall.SockaddrUnix{Name:"long_endpoint"}
 errline("cached first",syscall.Bind(first,cached))
 cached.Name="x"
 errline("cached short",syscall.Bind(second,cached))
 _,x:=os.Lstat("x")
 _,xo:=os.Lstat("xo")
 fmt.Println("actual cached paths",x==nil,xo==nil)
 for _,fd:=range []int{first,second,client,server} { errline("owned close",syscall.Close(fd)) }
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
