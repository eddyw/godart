package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNativeFileAllocation(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("File allocation requires Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/fallocate\n\ngo 1.27\n")
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

func size(path string) {
 info,err:=os.Stat(path)
 if err!=nil { panic(err) }
 fmt.Println("actual size",info.Size())
}

func main() {
 dir,err:=os.MkdirTemp("","fallocate-contract")
 if err!=nil { panic(err) }
 defer os.RemoveAll(dir)
 path:=filepath.Join(dir,"allocated")
 initial:=make([]byte,257)
 for i:=range initial { initial[i]=byte(i) }
 if err:=os.WriteFile(path,initial,0600);err!=nil { panic(err) }
 fd,err:=syscall.Open(path,syscall.O_RDWR,0)
 if err!=nil { panic(err) }
 errline("existing",syscall.Fallocate(fd,0,16,32))
 data,err:=os.ReadFile(path)
 if err!=nil { panic(err) }
 fmt.Println("existing bytes",bytes.Equal(data,initial))
 size(path)
 errline("extend",syscall.Fallocate(fd,0,300,17))
 data,err=os.ReadFile(path)
 if err!=nil { panic(err) }
 fmt.Println("prefix",bytes.Equal(data[:len(initial)],initial))
 fmt.Println("extension zero",bytes.Equal(data[len(initial):],make([]byte,len(data)-len(initial))))
 size(path)
 duplicate,err:=syscall.Dup(fd)
 if err!=nil { panic(err) }
 errline("duplicate allocation",syscall.Fallocate(duplicate,0,340,17))
 size(path)
 errline("duplicate close",syscall.Close(duplicate))
 offset:=int64(1)<<33|19
 errline("large offset",syscall.Fallocate(fd,0,offset,17))
 size(path)
 tail:=make([]byte,32)
 n,err:=syscall.Pread(fd,tail,offset)
 fmt.Println("actual tail",n)
 errline("pread",err)
 if n>=0 { fmt.Println("tail zero",bytes.Equal(tail[:n],make([]byte,n))) }
 errline("restore size",syscall.Ftruncate(fd,int64(len(initial))))
 for _,input:=range [][3]int64{{0,-1,17},{0,0,0},{0,0,-1},{int64(^uint32(0)),0,17}} {
  errline("invalid",syscall.Fallocate(fd,uint32(input[0]),input[1],input[2]))
  size(path)
 }
 errline("invalid descriptor",syscall.Fallocate(-1,0,0,17))
 readonly,err:=syscall.Open(path,syscall.O_RDONLY,0)
 if err!=nil { panic(err) }
 errline("readonly",syscall.Fallocate(readonly,0,0,17))
 errline("readonly close",syscall.Close(readonly))
 data,err=os.ReadFile(path)
 if err!=nil { panic(err) }
 fmt.Println("retained bytes",bytes.Equal(data,initial))
 errline("owned close",syscall.Close(fd))
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
