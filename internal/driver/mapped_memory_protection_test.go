package driver_test

import (
	"fmt"
	"github.com/eddyw/godart/internal/driver"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestMappedMemoryProtection(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native mappings require Darwin or Linux")
	}
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/mappedprotection\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"os";"syscall")
func must(e error){if e!=nil{panic(e)}}
func protection(label string,data []byte,prot int){err:=syscall.Mprotect(data,prot);code,typed:=err.(syscall.Errno);fmt.Println(label,int(code),typed)}
func main(){
 if len(os.Args)>1 {
  if os.Args[1]=="unavailable" {
   data:=[]byte{3,5,7};alias:=data
   for i,b:=range [][]byte{nil,{},data,data[1:]} {
    err:=syscall.Mprotect(b,syscall.PROT_READ);code,typed:=err.(syscall.Errno)
    fmt.Println("unavailable",i,int(code),typed,errors.Is(err,errors.ErrUnsupported),alias[0]==3&&alias[1]==5&&alias[2]==7)
   }
   return
  }
  var limit syscall.Rlimit;must(syscall.Getrlimit(syscall.RLIMIT_CORE,&limit));limit.Cur=0;must(syscall.Setrlimit(syscall.RLIMIT_CORE,&limit))
  data,err:=syscall.Mmap(-1,0,syscall.Getpagesize(),syscall.PROT_READ|syscall.PROT_WRITE,syscall.MAP_PRIVATE|syscall.MAP_ANON);must(err)
  prot:=syscall.PROT_NONE;if os.Args[1]=="write"{prot=syscall.PROT_READ};must(syscall.Mprotect(data,prot))
  fmt.Println("protected")
  if os.Args[1]=="write"{data[0]=11}else{fmt.Println("read",data[0])}
  fmt.Println("accessible")
  return
 }

 page:=syscall.Getpagesize();data,err:=syscall.Mmap(-1,0,3*page,syscall.PROT_READ|syscall.PROT_WRITE,syscall.MAP_PRIVATE|syscall.MAP_ANON);must(err);defer syscall.Munmap(data)
 data[0]=11;data[page]=17;data[2*page]=23;alias:=data
 protection("read",data[:page],syscall.PROT_READ);fmt.Println("read bytes",data[0]==11,alias[0]==11)
 protection("none",data[page:2*page],syscall.PROT_NONE);data[2*page]=29;fmt.Println("third",alias[2*page]==29)
 protection("restore read",data[:page],syscall.PROT_READ|syscall.PROT_WRITE);protection("restore none",data[page:2*page],syscall.PROT_READ|syscall.PROT_WRITE);fmt.Println("retained",data[0]==11,data[page]==17,data[2*page]==29)
 protection("unaligned",data[1:page],syscall.PROT_READ|syscall.PROT_WRITE)
 protection("invalid",data[:page],-1)
 protection("prot width",data[:page],(1<<32)|syscall.PROT_READ|syscall.PROT_WRITE)
 protection("short page",data[page:page+1],syscall.PROT_READ);protection("restore short",data[page:page+1],syscall.PROT_READ|syscall.PROT_WRITE)
 dir,err:=os.MkdirTemp("","mprotect");must(err);defer os.RemoveAll(dir);must(os.WriteFile(dir+"/data",make([]byte,page),0600));fd,err:=syscall.Open(dir+"/data",syscall.O_RDONLY,0);must(err);defer syscall.Close(fd)
 file,err:=syscall.Mmap(fd,0,page,syscall.PROT_READ,syscall.MAP_SHARED);must(err);defer syscall.Munmap(file);protection("read-only shared",file,syscall.PROT_READ|syscall.PROT_WRITE);fmt.Println("file retained",file[0]==0)
}
`)
	native := filepath.Join(t.TempDir(), "native")
	run(t, dir, "go", "build", "-o", native, ".")
	want := upstreamOutcome(t, dir, os.Environ(), runLimit, native)
	invalid, width, typed := "22", "0", "false"
	if runtime.GOOS == "darwin" {
		invalid = "45"
	}
	if runtime.GOOS == "linux" && runtime.GOARCH == "arm64" {
		width, typed = "22", "true"
	}
	expected := strings.NewReplacer("INVALID", invalid, "WIDTH", width, "TYPED", typed).Replace("read 0 false\nread bytes true true\nnone 0 false\nthird true\nrestore read 0 false\nrestore none 0 false\nretained true true true\nunaligned 22 true\ninvalid INVALID true\nprot width WIDTH TYPED\nshort page 0 false\nrestore short 0 false\nread-only shared 13 true\nfile retained true")
	if want.Exit != 0 || want.Text != expected {
		t.Fatalf("original native protection: %+v", want)
	}
	t.Log("original SDK Mprotect", want.Text)
	for _, mode := range []string{"read", "write"} {
		got := upstreamOutcome(t, dir, os.Environ(), runLimit, native, mode)
		if got.Exit != 2 || !strings.HasPrefix(got.Text, "protected\n") || !strings.Contains(got.Text, "unexpected fault address") || strings.Contains(got.Text, "accessible") {
			t.Fatalf("original protection fault %s: %+v", mode, got)
		}
		t.Log("original native protection fault", mode)
	}
	for _, goos := range []string{"darwin", "linux"} {
		t.Run(goos, func(t *testing.T) {
			t.Setenv("GOOS", goos)
			code := 38
			if goos == "darwin" {
				code = 78
			}
			unavailable := ""
			for i := 0; i < 4; i++ {
				if i > 0 {
					unavailable += "\n"
				}
				unavailable += fmt.Sprintf("unavailable %d %d true true true", i, code)
			}
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
								compilerInfo := filepath.Join(out, "resident-compiler.json")
								if target == "vm" {
									t.Cleanup(func() {
										run(t, out, "dart", "compilation-server", "shutdown", "--resident-compiler-info-file="+compilerInfo)
									})
								} else {
									run(t, out, "dart", "compile", "exe", "bin/main.dart", "-o", executable)
								}
								for _, mode := range []string{"normal", "unavailable", "read", "write"} {
									t.Run(mode, func(t *testing.T) {
										args := []string{}
										if mode != "normal" {
											args = append(args, mode)
										}
										var got programResult
										if target == "vm" {
											got = upstreamOutcome(t, dir, os.Environ(), runLimit, "dart", append([]string{"run", "--resident", "--quiet", "--verbosity=error", "--resident-compiler-info-file=" + compilerInfo, filepath.Join(out, "bin/main.dart")}, args...)...)
										} else {
											got = upstreamOutcome(t, dir, os.Environ(), runLimit, executable, args...)
										}
										switch mode {
										case "normal":
											if got != want {
												t.Fatalf("mapped protection differs: Go=%+v Gart=%+v", want, got)
											}
										case "unavailable":
											if got.Exit != 0 || got.Text != unavailable {
												t.Fatalf("native pointer limit: %+v", got)
											}
										default:
											signal := "si_signo=Segmentation fault"
											if runtime.GOOS == "darwin" {
												signal = "si_signo=Bus error"
											}
											if got.Exit != -1 || !strings.HasPrefix(got.Text, "protected\n") || !strings.Contains(got.Text, "===== CRASH =====") || !strings.Contains(got.Text, signal) || strings.Contains(got.Text, "accessible") {
												t.Fatalf("native protected access %s: %+v", mode, got)
											}
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
								t.Fatalf("web protection differs: %+v", got)
							}
						})
					}
				})
			}
		})
	}
}
