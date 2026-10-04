package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNativeSyscallOpenat(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Raw Openat requires Linux platform types")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/rawopenat\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
 "fmt"
 "os"
 "path/filepath"
 "syscall"
)

func opened(label string, parent int, path string, flags int, mode uint32) (int,error) {
 fd,err := syscall.Openat(parent,path,flags,mode)
 code,typed := err.(syscall.Errno)
 fmt.Printf("%s negative=%t success=%t errno=%d typed=%t\n",label,fd<0,err==nil,code,typed)
 return fd,err
}

func metadata(fd int) {
 var st syscall.Stat_t
 if err:=syscall.Fstat(fd,&st);err!=nil { panic(err) }
 fmt.Printf("metadata directory=%t fifo=%t mode=%o size=%d\n",st.Mode&syscall.S_IFMT==syscall.S_IFDIR,st.Mode&syscall.S_IFMT==syscall.S_IFIFO,st.Mode&07777,st.Size)
}

func read(fd int) {
 data:=make([]byte,20)
 for i:=range data { data[i]=0xcd }
 n,err:=syscall.Read(fd,data[2:18])
 code,typed:=err.(syscall.Errno)
 fmt.Printf("read count=%d errno=%d typed=%t bytes=%x\n",n,code,typed,data)
}

func close(fd int) {
 err:=syscall.Close(fd)
 code,typed:=err.(syscall.Errno)
 fmt.Printf("close errno=%d typed=%t\n",code,typed)
}

func main() {
 root,err:=os.MkdirTemp("","raw-openat-")
 if err!=nil { panic(err) }
 moved:=root+"-moved"
 defer os.RemoveAll(root)
 defer os.RemoveAll(moved)
 parent,err:=syscall.Open(root,syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_CLOEXEC,0)
 if err!=nil { panic(err) }
 fd,err:=opened("create",parent,"file",syscall.O_CREAT|syscall.O_EXCL|syscall.O_RDWR,0640)
 if err!=nil { panic(err) }
 n,err:=syscall.Write(fd,[]byte{'a',0,255,'z'})
 fmt.Println("write",n,err==nil)
 offset,err:=syscall.Seek(fd,0,0)
 fmt.Println("seek",offset,err==nil)
 read(fd)
 metadata(fd)
 close(fd)
 fd,err=opened("append",parent,"file",syscall.O_WRONLY|syscall.O_APPEND,0777)
 if err!=nil { panic(err) }
 offset,err=syscall.Seek(fd,0,0)
 fmt.Println("append seek",offset,err==nil)
 n,err=syscall.Write(fd,[]byte{'X'})
 fmt.Println("append write",n,err==nil)
 metadata(fd)
 close(fd)
 if err:=os.Symlink("file",filepath.Join(root,"link"));err!=nil { panic(err) }
 for _,test:=range []struct{name,path string;flags int}{
  {"read-only","file",syscall.O_RDONLY},
  {"high-flags","file",syscall.O_RDONLY|1<<40},
  {"directory",".",syscall.O_RDONLY|syscall.O_DIRECTORY},
  {"parent-directory","..",syscall.O_RDONLY|syscall.O_DIRECTORY},
  {"symlink","link",syscall.O_RDONLY},
  {"path-only","file",0x200000},
 } {
  fd,err=opened(test.name,parent,test.path,test.flags,0)
  if err!=nil { panic(err) }
  if test.name!="parent-directory" { metadata(fd) }
  if test.name!="directory" && test.name!="parent-directory" { read(fd) }
  close(fd)
 }
 for _,test:=range []struct{name,path string;flags int}{
  {"exclusive-existing","file",syscall.O_CREAT|syscall.O_EXCL|syscall.O_RDWR},
  {"nofollow","link",syscall.O_RDONLY|syscall.O_NOFOLLOW},
  {"directory-file","file",syscall.O_RDONLY|syscall.O_DIRECTORY},
  {"missing","missing",syscall.O_RDONLY},
  {"empty","",syscall.O_RDONLY},
  {"nul","file\x00",syscall.O_RDONLY},
  {"file-child","file/child",syscall.O_RDONLY},
  {"trailing-slash","file/",syscall.O_RDONLY},
 } {
  fd,err=opened(test.name,parent,test.path,test.flags,0)
  if err==nil { close(fd) }
 }
 for _,name:=range []string{"unicode-世界","raw-\xff"} {
  fd,err=opened("raw-name-create",parent,name,syscall.O_CREAT|syscall.O_EXCL|syscall.O_RDWR,uint32(1<<31)|0600)
  if err!=nil { panic(err) }
  n,err=syscall.Write(fd,[]byte{0,128,255})
  fmt.Println("raw-name write",n,err==nil)
  metadata(fd)
  close(fd)
  fd,err=opened("raw-name-read",parent,name,syscall.O_RDONLY,0)
  if err!=nil { panic(err) }
  read(fd)
  close(fd)
 }
 if err:=syscall.Mkfifo(filepath.Join(root,"fifo"),0600);err!=nil { panic(err) }
 fd,err=opened("nonblocking-fifo",parent,"fifo",syscall.O_RDONLY|syscall.O_NONBLOCK,0)
 if err!=nil { panic(err) }
 metadata(fd)
 read(fd)
 close(fd)
 fileParent,err:=syscall.Open(filepath.Join(root,"file"),syscall.O_RDONLY,0)
 if err!=nil { panic(err) }
 fd,err=opened("regular-parent",fileParent,"child",syscall.O_RDONLY,0)
 if err==nil { close(fd) }
 close(fileParent)
 fd,err=opened("invalid-parent",-1,"file",syscall.O_RDONLY,0)
 if err==nil { close(fd) }
 fd,err=opened("invalid-parent-nul",-1,"file\x00",syscall.O_RDONLY,0)
 if err==nil { close(fd) }
 fd,err=opened("absolute-invalid-parent",-1,filepath.Join(root,"file"),syscall.O_RDONLY,0)
 if err!=nil { panic(err) }
 read(fd)
 close(fd)
 if err:=os.Rename(root,moved);err!=nil { panic(err) }
 fd,err=opened("renamed-parent",parent,"after",syscall.O_CREAT|syscall.O_EXCL|syscall.O_RDWR,0600)
 if err!=nil { panic(err) }
 close(fd)
 _,err=os.Stat(filepath.Join(moved,"after"))
 fmt.Println("renamed entry",err==nil)
 fd,err=opened("truncate",parent,"file",syscall.O_RDWR|syscall.O_TRUNC,0)
 if err!=nil { panic(err) }
 metadata(fd)
 read(fd)
 close(fd)
 close(parent)
 fd,err=opened("closed-parent",parent,"file",syscall.O_RDONLY,0)
 if err==nil { close(fd) }
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
