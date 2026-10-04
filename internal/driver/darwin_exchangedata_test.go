package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDarwinExchangeData(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Exchangedata requires macOS")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/accept\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"os";"path/filepath";"syscall")
func must(err error){if err!=nil{panic(err)}}
func result(label string,err error){if err==nil{fmt.Println(label,0);return};code,ok:=err.(syscall.Errno);if !ok{panic("non-errno")};fmt.Println(label,int(code))}
func contents(label,path string){data,err:=os.ReadFile(path);must(err);fmt.Println(label,data)}
func main(){
 if len(os.Args)>1&&os.Args[1]=="unsupported"{
  valid:=true
  for i,err:=range []error{syscall.Exchangedata("first","second",0),syscall.Mknod("fifo",syscall.S_IFIFO|0600,0)}{code,typed:=err.(syscall.Errno);unsupported:=errors.Is(err,errors.ErrUnsupported);fmt.Println("unavailable",i,int(code),typed,unsupported);valid=valid&&typed&&code==syscall.ENOSYS&&unsupported}
  if !valid{panic("Darwin ENOSYS mismatch")};return
 }

 dir,err:=os.MkdirTemp("","exchangedata");must(err);defer os.RemoveAll(dir);first,second:=filepath.Join(dir,"first"),filepath.Join(dir,"second")
 must(os.WriteFile(first,[]byte{0,255,17,19},0600));must(os.WriteFile(second,[]byte{23,29,31},0600))
 fd,err:=syscall.Open(first,syscall.O_RDONLY,0);must(err);duplicate,err:=syscall.Dup(fd);must(err);must(syscall.Close(fd));defer syscall.Close(duplicate)
 for i,options:=range []int{0,1,2,-1,1<<32,1<<32|1,1<<63-1}{result(fmt.Sprint("options ",i),syscall.Exchangedata(first,second,options));contents("first",first);contents("second",second)}
 link:=filepath.Join(dir,"link");must(os.Symlink("first",link));dangling:=filepath.Join(dir,"dangling");must(os.Symlink("missing",dangling))
 for i,pair:=range [][2]string{{first,first},{link,second},{dangling,second},{dir,second},{filepath.Join(dir,"absent"),second},{first,filepath.Join(dir,"absent")},{"",second},{first,""},{"a\x00b",second},{first,"a\x00b"},{"a\x00b","c\x00d"},{filepath.Join(dir,string([]byte{255})),second}}{result(fmt.Sprint("paths ",i),syscall.Exchangedata(pair[0],pair[1],0))}
 contents("final first",first);contents("final second",second);data:=[]byte{37,41,43,47,53,59,61,67};n,err:=syscall.Read(duplicate,data[2:6]);must(err);fmt.Println("owned descriptor",n,data)
}
`)
	want := runProgram(t, dir, "go", "run", ".")
	t.Log(want)
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
	for _, target := range []string{"js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			got := runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{"unsupported"})
			if got.Exit != 0 || got.Text != "unavailable 0 78 true true\nunavailable 1 78 true true" {
				t.Fatalf("Darwin unavailable differs: %+v", got)
			}
		})
	}

}
