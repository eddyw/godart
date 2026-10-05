package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLinuxMappedMadvise(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/mappedadvice\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("bytes";"errors";"fmt";"os";"syscall")
func must(e error){if e!=nil{panic(e)}}
func advice(label string,data []byte,value int){e:=syscall.Madvise(data,value);code,typed:=e.(syscall.Errno);fmt.Println(label,int(code),typed)}
func main(){
 if len(os.Args)>1 {
  if os.Args[1]=="unavailable" {
   data:=[]byte{3,5,7};alias:=data
   for i,b:=range [][]byte{nil,{},data,data[1:]} {
    err:=syscall.Madvise(b,syscall.MADV_NORMAL);code,typed:=err.(syscall.Errno)
    fmt.Println("unavailable",i,int(code),typed,errors.Is(err,errors.ErrUnsupported),alias[0]==3&&alias[1]==5&&alias[2]==7)
   }
   return
  }
  data,err:=syscall.Mmap(-1,0,syscall.Getpagesize(),syscall.PROT_READ|syscall.PROT_WRITE,syscall.MAP_PRIVATE|syscall.MAP_ANON);must(err);must(syscall.Munmap(data))
  err=syscall.Madvise(data,syscall.MADV_NORMAL);code,typed:=err.(syscall.Errno)
  fmt.Println("released",int(code),typed,errors.Is(err,errors.ErrUnsupported))
  return
 }

 page:=syscall.Getpagesize();data,e:=syscall.Mmap(-1,0,3*page,syscall.PROT_READ|syscall.PROT_WRITE,syscall.MAP_PRIVATE|syscall.MAP_ANON);must(e);defer syscall.Munmap(data);for i:=range data{data[i]=byte(i*17+3)};alias:=data;copy:=append([]byte(nil),data...)
 advice("normal",data,syscall.MADV_NORMAL);fmt.Println("normal bytes",bytes.Equal(data,copy))
 advice("unaligned",data[1:page],syscall.MADV_NORMAL);advice("invalid",data[:page],-1);advice("width",data[:page],(1<<32)|syscall.MADV_NORMAL);fmt.Println("errors retain",bytes.Equal(data,copy))
 advice("discard middle",data[page:2*page],syscall.MADV_DONTNEED);fmt.Println("middle bytes",bytes.Equal(data[:page],copy[:page]),bytes.Equal(alias[page:2*page],make([]byte,page)),bytes.Equal(data[2*page:],copy[2*page:]))
 alias[page]=91;advice("discard short",data[page:page+1],syscall.MADV_DONTNEED);fmt.Println("short bytes",bytes.Equal(data[page:2*page],make([]byte,page)),alias[page]==0)
 dir,e:=os.MkdirTemp("","madvise");must(e);defer os.RemoveAll(dir);payload:=make([]byte,page);for i:=range payload{payload[i]=byte(i*13+5)};must(os.WriteFile(dir+"/data",payload,0600));fd,e:=syscall.Open(dir+"/data",syscall.O_RDONLY,0);must(e)
 file,e:=syscall.Mmap(fd,0,page,syscall.PROT_READ|syscall.PROT_WRITE,syscall.MAP_PRIVATE);must(e);must(syscall.Close(fd));defer syscall.Munmap(file);file[0]^=255;advice("discard file",file,syscall.MADV_DONTNEED);got,e:=os.ReadFile(dir+"/data");must(e);fmt.Println("file bytes",bytes.Equal(file,payload),bytes.Equal(got,payload))
}
`)
	want := programResult{Exit: 0, Text: "normal 0 false\nnormal bytes true\nunaligned 22 true\ninvalid 22 true\nwidth 0 false\nerrors retain true\ndiscard middle 0 false\nmiddle bytes true true true\ndiscard short 0 false\nshort bytes true true\ndiscard file 0 false\nfile bytes true true"}
	if runtime.GOOS == "linux" {
		native := filepath.Join(t.TempDir(), "native")
		run(t, dir, "go", "build", "-o", native, ".")
		got := upstreamOutcome(t, dir, os.Environ(), runLimit, native)
		if got != want {
			t.Fatalf("original native advice: %+v", got)
		}
		t.Log("original SDK Madvise", got.Text)
	}
	t.Setenv("GOOS", "linux")
	unavailable := "unavailable 0 38 true true true\nunavailable 1 38 true true true\nunavailable 2 38 true true true\nunavailable 3 38 true true true"
	for _, arch := range []string{"amd64", "arm64"} {
		t.Run(arch, func(t *testing.T) {
			t.Setenv("GOARCH", arch)
			out := t.TempDir()
			if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
				t.Fatal(err)
			}
			prepareDart(t, out)
			if runtime.GOOS == "linux" {
				for _, target := range []string{"vm", "aot"} {
					t.Run(target, func(t *testing.T) {
						executable := filepath.Join(out, "mapped.exe")
						if target == "aot" {
							run(t, out, "dart", "compile", "exe", "bin/main.dart", "-o", executable)
						}
						for _, mode := range []string{"normal", "unavailable", "released"} {
							t.Run(mode, func(t *testing.T) {
								args := []string{}
								if mode != "normal" {
									args = append(args, mode)
								}
								var got programResult
								if target == "vm" {
									got = upstreamOutcome(t, dir, os.Environ(), runLimit, "dart", append([]string{filepath.Join(out, "bin/main.dart")}, args...)...)
								} else {
									got = upstreamOutcome(t, dir, os.Environ(), runLimit, executable, args...)
								}
								expected := want
								if mode == "unavailable" {
									expected = programResult{Exit: 0, Text: unavailable}
								}
								if mode == "released" {
									expected = programResult{Exit: 0, Text: "released 38 true true"}
								}
								if got != expected {
									t.Fatalf("mapped advice %s differs: want=%+v Gart=%+v", mode, expected, got)
								}
							})
						}
					})
				}
			}
			for _, target := range []string{"js", "wasm"} {
				t.Run(target, func(t *testing.T) {
					got := runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{"unavailable"})
					if got.Exit != 0 || got.Text != unavailable {
						t.Fatalf("web advice differs: %+v", got)
					}
				})
			}
		})
	}
}
