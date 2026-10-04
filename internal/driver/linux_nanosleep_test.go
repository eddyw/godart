package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLinuxNanosleep(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Nanosleep requires Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/accept\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"syscall";"time")
func result(label string,err error){if err==nil{fmt.Println(label,0);return};code,ok:=err.(syscall.Errno);if !ok{panic("non-errno")};fmt.Println(label,int(code))}
func main(){
 for i,request:=range []syscall.Timespec{{0,0},{0,1},{0,1000000},{-1,0},{0,-1},{0,1000000000},{1<<62,-1},{-1<<62,0}}{
  before:=request;left:=syscall.Timespec{Sec:71,Nsec:73};old:=left;start:=time.Now();err:=syscall.Nanosleep(&request,&left);result(fmt.Sprint("request ",i),err);fmt.Println("retained",request==before,left==old);if i==2&&err==nil{fmt.Println("elapsed",time.Since(start)>=time.Millisecond)}
 }
 left:=syscall.Timespec{Sec:79,Nsec:83};old:=left;result("nil request",syscall.Nanosleep(nil,&left));fmt.Println("nil retained",left==old);result("both nil",syscall.Nanosleep(nil,nil))
 request:=syscall.Timespec{};result("nil output",syscall.Nanosleep(&request,nil));fmt.Println("nil input retained",request)
 values:=[3]syscall.Timespec{{Sec:89,Nsec:97},{Sec:0,Nsec:1},{Sec:101,Nsec:103}};before:=values;result("alias",syscall.Nanosleep(&values[1],&values[1]));fmt.Println("alias retained",values==before)
 values[1]=syscall.Timespec{Sec:0,Nsec:-1};before=values;result("invalid alias",syscall.Nanosleep(&values[1],&values[1]));fmt.Println("invalid alias retained",values==before)
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
