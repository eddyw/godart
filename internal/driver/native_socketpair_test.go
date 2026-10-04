package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNativeSocketpair(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("Native socket pairs require Linux or macOS")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/socketpair\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
 "fmt"
 "os"
 "runtime"
 "strconv"
 "strings"
 "syscall"
)

func read(label string,fd int) {
 data:=make([]byte,12)
 for i:=range data { data[i]=0xcd }
 n,err:=syscall.Read(fd,data[2:10])
 code,typed:=err.(syscall.Errno)
 fmt.Printf("read %s count=%d errno=%d typed=%t bytes=%x\n",label,n,code,typed,data)
}

func write(label string,fd int,data []byte) {
 n,err:=syscall.Write(fd,data)
 code,typed:=err.(syscall.Errno)
 fmt.Printf("write %s count=%d errno=%d typed=%t\n",label,n,code,typed)
}

func close(label string,fd int) {
 err:=syscall.Close(fd)
 code,typed:=err.(syscall.Errno)
 fmt.Printf("close %s errno=%d typed=%t\n",label,code,typed)
}

func flags(fd int) {
 if runtime.GOOS!="linux" { return }
 data,err:=os.ReadFile("/proc/self/fdinfo/"+strconv.Itoa(fd))
 if err!=nil { panic(err) }
 for _,line:=range strings.Split(string(data),"\n") {
  name,value,ok:=strings.Cut(line,":")
  if ok && name=="flags" {
   bits,err:=strconv.ParseUint(strings.TrimSpace(value),8,64)
   if err!=nil { panic(err) }
   fmt.Printf("flags nonblock=%t cloexec=%t\n",bits&syscall.O_NONBLOCK!=0,bits&syscall.O_CLOEXEC!=0)
   return
  }
 }
 panic("missing descriptor flags")
}

func main() {
 types:=[]int{syscall.SOCK_STREAM,syscall.SOCK_DGRAM,syscall.SOCK_SEQPACKET}
 if runtime.GOOS=="linux" { types=append(types,syscall.SOCK_STREAM|0x800,syscall.SOCK_DGRAM|0x80000|0x800) }
 for _,kind:=range types {
  pair,err:=syscall.Socketpair(syscall.AF_UNIX,kind,0)
  code,typed:=err.(syscall.Errno)
  fmt.Printf("pair type=%d errno=%d typed=%t\n",kind,code,typed)
  if err!=nil { fmt.Println("error pair",pair);continue }
  fmt.Println("descriptors",pair[0]>=0,pair[1]>=0,pair[0]!=pair[1])
  for _,fd:=range pair {
   flags(fd)
   value,err:=syscall.GetsockoptInt(fd,syscall.SOL_SOCKET,syscall.SO_TYPE)
   code,typed:=err.(syscall.Errno)
   fmt.Printf("type value=%d errno=%d typed=%t\n",value,code,typed)
   if err:=syscall.SetNonblock(fd,true);err!=nil { panic(err) }
  }
  read("empty",pair[0])
  write("first",pair[0],[]byte{0,128,255,'A'})
  read("peer",pair[1])
  write("reverse",pair[1],[]byte{'B',0,255})
  read("first",pair[0])
  duplicate,err:=syscall.Dup(pair[0])
  if err!=nil { panic(err) }
  close("original",pair[0])
  write("duplicate",duplicate,[]byte{'D'})
  read("duplicate-peer",pair[1])
  close("duplicate",duplicate)
  if kind&15==syscall.SOCK_STREAM || kind&15==syscall.SOCK_SEQPACKET { read("eof",pair[1]) }
  close("peer",pair[1])
  read("closed",pair[1])
 }
 for _,test:=range []struct{domain,kind,protocol int}{
  {-1,syscall.SOCK_STREAM,0},{syscall.AF_INET,syscall.SOCK_STREAM,0},
  {syscall.AF_UNIX,0,0},{syscall.AF_UNIX,syscall.SOCK_RAW,0},
  {syscall.AF_UNIX,syscall.SOCK_STREAM,-1},{syscall.AF_UNIX,syscall.SOCK_STREAM,1},
  {syscall.AF_UNIX,syscall.SOCK_STREAM|0x40000000,0},
 } {
  pair,err:=syscall.Socketpair(test.domain,test.kind,test.protocol)
  code,typed:=err.(syscall.Errno)
  fmt.Printf("invalid domain=%d type=%d protocol=%d errno=%d typed=%t\n",test.domain,test.kind,test.protocol,code,typed)
  if err!=nil { fmt.Println("error pair",pair) } else { close("unexpected-first",pair[0]);close("unexpected-second",pair[1]) }
 }
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
