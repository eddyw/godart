package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNativeMemoryMap(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("Native memory maps require Linux or macOS")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/memorymap\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
 "bytes"
 "fmt"
 "os"
 "path/filepath"
 "syscall"
)

func errline(label string,err error) {
 code,typed:=err.(syscall.Errno)
 fmt.Println(label,code,typed)
}

func main() {
 page:=syscall.Getpagesize()
 for _,length:=range []int{-1,0,1,page+5} {
  data,err:=syscall.Mmap(-1,0,length,syscall.PROT_READ|syscall.PROT_WRITE,syscall.MAP_ANON|syscall.MAP_PRIVATE)
  fmt.Println("anonymous",length,data==nil,len(data),cap(data))
  errline("mmap",err)
  if err!=nil { continue }
  fmt.Println("initial zero",bytes.Equal(data,make([]byte,length)))
  data[0]=128
  data[length-1]=255
  tail:=data[length/2:]
  tail[0]=42
  fmt.Println("alias",data[length/2]==42,data[length-1])
  errline("short header",syscall.Munmap(data[:length-1]))
  errline("short capacity",syscall.Munmap(data[:length-1:length-1]))
  errline("offset header",syscall.Munmap(data[1:]))
  copied:=append([]byte(nil),data...)
  errline("copied storage",syscall.Munmap(copied))
  fmt.Println("invalid calls retain memory",data[length/2]==42)
  errline("unmap",syscall.Munmap(data))
  errline("unmap again",syscall.Munmap(data))
 }
 errline("nil unmap",syscall.Munmap(nil))
 errline("empty unmap",syscall.Munmap([]byte{}))
 first,err:=syscall.Mmap(-1,0,page,syscall.PROT_READ|syscall.PROT_WRITE,syscall.MAP_ANON|syscall.MAP_PRIVATE)
 if err!=nil { panic(err) }
 second,err:=syscall.Mmap(-1,0,page,syscall.PROT_READ|syscall.PROT_WRITE,syscall.MAP_ANON|syscall.MAP_PRIVATE)
 if err!=nil { panic(err) }
 first[0]=23
 second[0]=42
 fmt.Println("independent maps",first[0],second[0])
 if err:=syscall.Munmap(first);err!=nil { panic(err) }
 fmt.Println("independent lifetime",second[0])
 if err:=syscall.Munmap(second);err!=nil { panic(err) }
 for _,input:=range [][3]int64{{-1,0,int64(page)},{-1,1,int64(page)}} {
  data,err:=syscall.Mmap(int(input[0]),input[1],int(input[2]),syscall.PROT_READ,syscall.MAP_PRIVATE)
  fmt.Println("invalid file",data==nil,len(data),cap(data))
  errline("invalid map",err)
 }
 dir,err:=os.MkdirTemp("","mmap-contract")
 if err!=nil { panic(err) }
 defer os.RemoveAll(dir)
 path:=filepath.Join(dir,"mapped")
 initial:=make([]byte,page*2+5)
 for i:=range initial { initial[i]=byte(i) }
 if err:=os.WriteFile(path,initial,0600);err!=nil { panic(err) }
 for _,flag:=range []int{syscall.MAP_SHARED,syscall.MAP_PRIVATE} {
  fd,err:=syscall.Open(path,syscall.O_RDWR,0)
  if err!=nil { panic(err) }
  data,err:=syscall.Mmap(fd,int64(page),page+5,syscall.PROT_READ|syscall.PROT_WRITE,flag)
  if err!=nil { panic(err) }
  if err:=syscall.Close(fd);err!=nil { panic(err) }
  want,err:=os.ReadFile(path)
  if err!=nil { panic(err) }
  fmt.Println("file offset",bytes.Equal(data,want[page:]))
  data[0]^=255
  data[len(data)-1]^=128
  changed:=append([]byte(nil),data...)
  if err:=syscall.Munmap(data);err!=nil { panic(err) }
  got,err:=os.ReadFile(path)
  if err!=nil { panic(err) }
  fmt.Println("file prefix",bytes.Equal(got[:page],want[:page]))
  fmt.Println("file shared",bytes.Equal(got[page:],changed),bytes.Equal(got,want))
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
