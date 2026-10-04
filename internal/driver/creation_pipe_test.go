package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNativePipeCreation(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("Native pipe requires Unix")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/creation\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"os";"runtime";"strconv";"strings";"syscall")
func must(err error){if err!=nil{panic(err)}}
func main(){
 for _,length:=range []int{0,1,3}{p:=[]int{17,19,23}[:length];before:=fmt.Sprint(p);err:=syscall.Pipe(p);fmt.Println("length",length,err==syscall.EINVAL,fmt.Sprint(p)==before)}
 storage:=[]int{31,0,0,37};p:=storage[1:3];must(syscall.Pipe(p));defer syscall.Close(p[0]);defer syscall.Close(p[1]);fmt.Println("caller",storage[0]==31,storage[3]==37,p[0]!=p[1],p[0]>=0,p[1]>=0)
 if runtime.GOOS=="linux"{for _,fd:=range p{data,err:=os.ReadFile(fmt.Sprintf("/proc/self/fdinfo/%d",fd));must(err);found:=false;for _,line:=range strings.Split(string(data),"\n"){if strings.HasPrefix(line,"flags:"){flags,err:=strconv.ParseUint(strings.TrimSpace(strings.TrimPrefix(line,"flags:")),8,64);must(err);fmt.Println("flags",flags&syscall.O_NONBLOCK==0,flags&syscall.O_CLOEXEC==0);found=true}};if !found{panic("flags absent")}}}
 duplicate,err:=syscall.Dup(p[0]);must(err);defer syscall.Close(duplicate)
 n,err:=syscall.Read(p[0],nil);must(err);fmt.Println("empty read",n)
 payload:=[]byte("ab\x00cd\xffef");n,err=syscall.Write(p[1],payload);must(err);fmt.Println("write",n)
 buffer:=[]byte{0xa5,0xa5,0xa5,0xa5,0xa5,0xa5};n,err=syscall.Read(duplicate,buffer[1:5]);must(err);fmt.Printf("offset %d %x\n",n,buffer)
 must(syscall.Close(p[0]));n,err=syscall.Read(duplicate,buffer[1:5]);must(err);fmt.Printf("duplicate %d %x\n",n,buffer[1:1+n])
 must(syscall.Close(p[1]));n,err=syscall.Read(duplicate,buffer[1:5]);must(err);fmt.Println("EOF",n)
 n,err=syscall.Write(p[1],[]byte("closed"));fmt.Println("closed",n,err==syscall.EBADF)
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
