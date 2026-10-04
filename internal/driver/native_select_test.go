package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNativeDescriptorSelect(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("Native descriptor selection requires Linux or macOS")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/select\n\ngo 1.27\n")
	source := `package main

import (
 "fmt"
 "os"
 "syscall"
)

func errline(label string,err error) {
 code,typed:=err.(syscall.Errno)
 fmt.Println(label,code,typed)
}

func set(fds ...int) *syscall.FdSet {
 result:=new(syscall.FdSet)
 for _,fd:=range fds { result.Bits[fd/32] |= int32(1)<<uint(fd%32) }
 return result
}

func has(set *syscall.FdSet,fd int) bool {
 return set.Bits[fd/32] & (int32(1)<<uint(fd%32)) != 0
}

func call(label string,nfd int,r,w,e *syscall.FdSet,tv *syscall.Timeval) {
 err:=syscall.Select(nfd,r,w,e,tv)
 errline(label,err)
 if tv!=nil { fmt.Println("timeout",tv.Sec,tv.Usec) }
}

func main() {
 reader,writer,err:=os.Pipe()
 if err!=nil { panic(err) }
 rfd,wfd:=int(reader.Fd()),int(writer.Fd())
 if err:=syscall.SetNonblock(rfd,true);err!=nil { panic(err) }
 if err:=syscall.SetNonblock(wfd,true);err!=nil { panic(err) }
 owned:=[]int{}
 for {
  fd,err:=syscall.Open("/dev/null",syscall.O_RDONLY,0)
  if err!=nil { panic(err) }
  owned=append(owned,fd)
  if fd>=70 { break }
 }
 high,err:=syscall.Dup(rfd)
 if err!=nil { panic(err) }
 for _,fd:=range owned { if err:=syscall.Close(fd);err!=nil { panic(err) } }
 for _,ready:=range []bool{false,true} {
  if ready {
   n,err:=syscall.Write(wfd,[]byte{0,128,255})
   fmt.Println("actual write",n)
   errline("write",err)
  }
  r,w,e:=set(rfd,high),set(wfd),set(rfd)
  call("select",high+1,r,w,e,&syscall.Timeval{})
  fmt.Println("readiness",has(r,rfd),has(r,high),has(w,wfd),has(e,rfd))
 }
 r:=set(rfd)
 call("nil timeout ready",rfd+1,r,nil,nil,nil)
 fmt.Println("ready nil timeout",has(r,rfd))
 shared:=set(rfd,wfd)
 call("aliased sets",high+1,shared,shared,nil,&syscall.Timeval{})
 fmt.Println("shared output",has(shared,rfd),has(shared,wfd))
 data:=make([]byte,8)
 n,err:=syscall.Read(rfd,data)
 fmt.Println("actual read",n,data[:n])
 errline("read",err)
 call("nil sets",0,nil,nil,nil,&syscall.Timeval{})
 call("invalid count",-1,nil,nil,nil,&syscall.Timeval{})
 call("invalid time",0,nil,nil,nil,&syscall.Timeval{Sec:-1})
 if err:=writer.Close();err!=nil { panic(err) }
 r=set(rfd,high)
 call("EOF ready",high+1,r,nil,nil,&syscall.Timeval{})
 fmt.Println("EOF flags",has(r,rfd),has(r,high))
 n,err=syscall.Read(high,data)
 fmt.Println("actual EOF",n)
 errline("read",err)
 if err:=syscall.Close(high);err!=nil { panic(err) }
 call("invalid descriptor",high+1,set(high),nil,nil,&syscall.Timeval{})
 errline("reader owned",reader.Close())
}
`
	if runtime.GOOS == "linux" {
		source = strings.ReplaceAll(source, "int32", "int64")
		source = strings.ReplaceAll(source, "/32", "/64")
		source = strings.ReplaceAll(source, "%32", "%64")
		source = strings.Replace(source, "err:=syscall.Select(nfd,r,w,e,tv)", `count,err:=syscall.Select(nfd,r,w,e,tv)
 fmt.Println("ready count",count)`, 1)
	}
	writeFile(t, filepath.Join(dir, "main.go"), source)
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

func TestNativeDescriptorSelectBounds(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("Native descriptor bounds require Linux or macOS")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/selectbounds\n\ngo 1.27\n")
	source := `package main
import (
 "fmt"
 "syscall"
)
func main() {
 for _,bound:=range []int{1025,1<<30,1<<32|1} {
  set:=syscall.FdSet{}
  set.Bits[0]=1
  before:=set
  timeval:=syscall.Timeval{Sec:123,Usec:456}
  timeout:=timeval
  err:=syscall.Select(bound,&set,&set,&set,&timeval)
  if err!=syscall.EINVAL || set!=before || timeval!=timeout { panic("invalid bound changed caller state") }
  fmt.Println("bound",bound,"retains caller state")
 }
}`
	if runtime.GOOS == "linux" {
		source = strings.Replace(source, "err:=syscall.Select(bound,&set,&set,&set,&timeval)", `count,err:=syscall.Select(bound,&set,&set,&set,&timeval)
  if count != -1 { panic("invalid bound result") }`, 1)
	}
	writeFile(t, filepath.Join(dir, "main.go"), source)
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	const want = "bound 1025 retains caller state\nbound 1073741824 retains caller state\nbound 4294967297 retains caller state"
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			var got programResult
			if target == "vm" {
				got = upstreamOutcome(t, dir, nil, runLimit, "dart", filepath.Join(out, "bin/main.dart"))
			} else {
				got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
			}
			if got.Exit != 0 || got.Text != want {
				t.Fatalf("%s: exit %d\nWant:\n%s\nGart:\n%s", target, got.Exit, want, got.Text)
			}
		})
	}
}
