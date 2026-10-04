package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLinuxSyncFileRange(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("SyncFileRange requires Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/accept\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"os";"path/filepath";"syscall")
func must(err error){if err!=nil{panic(err)}}
func result(label string,err error){if err==nil{fmt.Println(label,0);return};code,ok:=err.(syscall.Errno);if !ok{panic("non-errno")};fmt.Println(label,int(code))}
func main(){
 dir,err:=os.MkdirTemp("","sync-range");must(err);defer os.RemoveAll(dir)
 fd,err:=syscall.Open(filepath.Join(dir,"file"),syscall.O_CREAT|syscall.O_RDWR,0600);must(err)
 bytes:=[]byte{0,255,17,19,23,29};n,err:=syscall.Write(fd,bytes);must(err);if n!=len(bytes){panic("write count")}
 _,err=syscall.Seek(fd,3,0);must(err)
 for flags:=0;flags<8;flags++{result(fmt.Sprint("flags ",flags),syscall.SyncFileRange(fd,0,6,flags))}
 for i,value:=range [][3]int64{{0,0,7},{2,3,7},{1<<33|19,17,7},{0,1<<33|23,0},{-1,1,0},{0,-1,0},{1<<63-1,1,0},{0,1,8},{0,1,-1}}{result(fmt.Sprint("range ",i),syscall.SyncFileRange(fd,value[0],value[1],int(value[2])))}
 position,err:=syscall.Seek(fd,0,1);must(err);got:=make([]byte,len(bytes));n,err=syscall.Pread(fd,got,0);must(err);fmt.Println("position bytes",position,n,got,bytes)
 wide:=int64(1<<33|19);n,err=syscall.Pwrite(fd,[]byte{31,255,37},wide);must(err);fmt.Println("wide write",n);result("wide sync",syscall.SyncFileRange(fd,wide,3,7));got=make([]byte,3);n,err=syscall.Pread(fd,got,wide);must(err);position,err=syscall.Seek(fd,0,1);must(err);fmt.Println("wide bytes position",n,got,position)
 duplicate,err:=syscall.Dup(fd);must(err);must(syscall.Close(fd));result("duplicate",syscall.SyncFileRange(duplicate,0,0,7));must(syscall.Close(duplicate));result("closed",syscall.SyncFileRange(duplicate,0,0,0))
 result("invalid",syscall.SyncFileRange(-1,0,0,0));result("invalid flags",syscall.SyncFileRange(-1,0,0,8))
 pair:=make([]int,2);must(syscall.Pipe(pair));result("pipe",syscall.SyncFileRange(pair[0],0,0,0));must(syscall.Close(pair[0]));must(syscall.Close(pair[1]))
 fd,err=syscall.Socket(syscall.AF_INET,syscall.SOCK_DGRAM,0);must(err);result("socket",syscall.SyncFileRange(fd,0,0,0));must(syscall.Close(fd))
 fd,err=syscall.Open(filepath.Join(dir,"file"),syscall.O_RDONLY,0);must(err);result("read-only",syscall.SyncFileRange(fd,0,6,7));must(syscall.Close(fd))
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
