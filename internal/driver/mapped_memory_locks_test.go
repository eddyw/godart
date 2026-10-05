package driver_test

import (
	"fmt"
	"github.com/eddyw/godart/internal/driver"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestMappedMemoryLocks(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native mapped locks require Darwin or Linux")
	}
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/mappedlocks\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("bytes";"errors";"fmt";"os";"runtime";"strconv";"strings";"syscall")
func must(err error){if err!=nil{panic(err)}}
func locked()int{if runtime.GOOS!="linux"{return 0};data,err:=os.ReadFile("/proc/self/status");must(err);for _,line:=range strings.Split(string(data),"\n"){fields:=strings.Fields(line);if len(fields)>1&&fields[0]=="VmLck:"{value,err:=strconv.Atoi(fields[1]);must(err);return value*1024}};panic("no VmLck")}
func pages(label string,base,page,want int){if runtime.GOOS=="linux"{fmt.Println(label,(locked()-base)/page==want)}}
func result(label string,err error){code,typed:=err.(syscall.Errno);fmt.Println(label,int(code),typed)}
func main(){
 if len(os.Args)>1 {
  if os.Args[1]=="unavailable" {
   data:=[]byte{3,5,7};alias:=data
   for _,operation:=range []string{"lock","unlock"}{for i,b:=range [][]byte{nil,{},data,data[1:]} {
    var err error;if operation=="lock"{err=syscall.Mlock(b)}else{err=syscall.Munlock(b)};code,typed:=err.(syscall.Errno)
    fmt.Println("unavailable",operation,i,int(code),typed,errors.Is(err,errors.ErrUnsupported),alias[0]==3&&alias[1]==5&&alias[2]==7)
   }}
   return
  }
  data,err:=syscall.Mmap(-1,0,syscall.Getpagesize(),syscall.PROT_READ|syscall.PROT_WRITE,syscall.MAP_PRIVATE|syscall.MAP_ANON);must(err);must(syscall.Munmap(data))
  for _,operation:=range []string{"lock","unlock"}{if operation=="lock"{err=syscall.Mlock(data)}else{err=syscall.Munlock(data)};code,typed:=err.(syscall.Errno);fmt.Println("released",operation,int(code),typed,errors.Is(err,errors.ErrUnsupported))}
  return
 }
page:=syscall.Getpagesize();data,err:=syscall.Mmap(-1,0,3*page,syscall.PROT_READ|syscall.PROT_WRITE,syscall.MAP_PRIVATE|syscall.MAP_ANON);must(err);defer syscall.Munmap(data);for i:=range data{data[i]=byte(i*17+3)};alias:=data;original:=append([]byte(nil),data...);base:=locked()
result("lock page",syscall.Mlock(data[:page]));pages("page state",base,page,1)
result("lock repeat",syscall.Mlock(data[:page]));pages("repeat state",base,page,1)
result("unlock page",syscall.Munlock(data[:page]));pages("unlock state",base,page,0)
result("lock short",syscall.Mlock(data[page+1:page+2]));pages("short state",base,page,1)
result("unlock short",syscall.Munlock(data[page+1:page+2]));pages("short unlock state",base,page,0)
result("lock boundary",syscall.Mlock(data[page-1:page+1]));pages("boundary state",base,page,2)
result("unlock first",syscall.Munlock(data[:page]));pages("first unlock state",base,page,1)
result("unlock second",syscall.Munlock(data[page:2*page]));pages("second unlock state",base,page,0)
result("unlock repeat",syscall.Munlock(data[page:2*page]));fmt.Println("bytes",bytes.Equal(alias,original))
memoryLimit:=8;if runtime.GOOS=="darwin"{memoryLimit=6};var limit syscall.Rlimit;must(syscall.Getrlimit(memoryLimit,&limit));low:=limit;low.Cur=0;must(syscall.Setrlimit(memoryLimit,&low));result("zero limit",syscall.Mlock(data[:page]));must(syscall.Setrlimit(memoryLimit,&limit));result("restored limit",syscall.Mlock(data[:page]));result("restored unlock",syscall.Munlock(data[:page]));pages("final state",base,page,0);fmt.Println("final bytes",bytes.Equal(data,original));must(syscall.Mprotect(data[:page],syscall.PROT_NONE));result("protected lock",syscall.Mlock(data[:page]));result("protected unlock",syscall.Munlock(data[:page]));must(syscall.Mprotect(data[:page],syscall.PROT_READ|syscall.PROT_WRITE));fmt.Println("protected bytes",bytes.Equal(data,original))
dir,err:=os.MkdirTemp("","mappedlock");must(err);defer os.RemoveAll(dir);payload:=make([]byte,page);for i:=range payload{payload[i]=byte(i*13+5)};must(os.WriteFile(dir+"/data",payload,0600));fd,err:=syscall.Open(dir+"/data",syscall.O_RDONLY,0);must(err)
file,err:=syscall.Mmap(fd,0,page,syscall.PROT_READ,syscall.MAP_SHARED);must(err);must(syscall.Close(fd));defer syscall.Munmap(file);fileAlias:=file
result("file lock",syscall.Mlock(file));pages("file state",base,page,1);result("file unlock",syscall.Munlock(file));pages("file unlock state",base,page,0);got,err:=os.ReadFile(dir+"/data");must(err);fmt.Println("file bytes",bytes.Equal(fileAlias,payload),bytes.Equal(got,payload))
}
`)
	native := filepath.Join(t.TempDir(), "native")
	run(t, dir, "go", "build", "-o", native, ".")
	want := upstreamOutcome(t, dir, os.Environ(), runLimit, native)
	expected := "lock page 0 false\nlock repeat 0 false\nunlock page 0 false\nlock short 0 false\nunlock short 0 false\nlock boundary 0 false\nunlock first 0 false\nunlock second 0 false\nunlock repeat 0 false\nbytes true\nzero limit 0 false\nrestored limit 0 false\nrestored unlock 0 false\nfinal bytes true\nprotected lock 0 false\nprotected unlock 0 false\nprotected bytes true\nfile lock 0 false\nfile unlock 0 false\nfile bytes true true"
	if runtime.GOOS == "linux" {
		expected = "lock page 0 false\npage state true\nlock repeat 0 false\nrepeat state true\nunlock page 0 false\nunlock state true\nlock short 0 false\nshort state true\nunlock short 0 false\nshort unlock state true\nlock boundary 0 false\nboundary state true\nunlock first 0 false\nfirst unlock state true\nunlock second 0 false\nsecond unlock state true\nunlock repeat 0 false\nbytes true\nzero limit 1 true\nrestored limit 0 false\nrestored unlock 0 false\nfinal state true\nfinal bytes true\nprotected lock 12 true\nprotected unlock 0 false\nprotected bytes true\nfile lock 0 false\nfile state true\nfile unlock 0 false\nfile unlock state true\nfile bytes true true"
	}
	if want.Exit != 0 || want.Text != expected {
		t.Fatalf("original native mapped locks: %+v", want)
	}
	t.Log("original SDK mapped locks", want.Text)
	for _, goos := range []string{"darwin", "linux"} {
		t.Run(goos, func(t *testing.T) {
			t.Setenv("GOOS", goos)
			code := 38
			if goos == "darwin" {
				code = 78
			}
			unavailable := ""
			for _, operation := range []string{"lock", "unlock"} {
				for i := 0; i < 4; i++ {
					if unavailable != "" {
						unavailable += "\n"
					}
					unavailable += fmt.Sprintf("unavailable %s %d %d true true true", operation, i, code)
				}
			}
			released := fmt.Sprintf("released lock %d true true\nreleased unlock %d true true", code, code)
			for _, arch := range []string{"amd64", "arm64"} {
				t.Run(arch, func(t *testing.T) {
					t.Setenv("GOARCH", arch)
					out := t.TempDir()
					if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
						t.Fatal(err)
					}
					prepareDart(t, out)
					if goos == runtime.GOOS {
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
											expected = programResult{Exit: 0, Text: released}
										}
										if got != expected {
											t.Fatalf("mapped locks %s differ: want=%+v Gart=%+v", mode, expected, got)
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
								t.Fatalf("web locks differ: %+v", got)
							}
						})
					}
				})
			}
		})
	}
}
