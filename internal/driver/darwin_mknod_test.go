package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDarwinMknod(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Mknod requires macOS")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/accept\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"os";"path/filepath";"syscall")
func must(err error){if err!=nil{panic(err)}}
func result(label string,err error){if err==nil{fmt.Println(label,0);return};code,ok:=err.(syscall.Errno);if !ok{panic("non-errno")};fmt.Println(label,int(code))}
func metadata(label,path string){var value syscall.Stat_t;must(syscall.Stat(path,&value));fmt.Printf("%s %o %d\n",label,value.Mode,value.Rdev)}
func main(){
 dir,err:=os.MkdirTemp("","darwin-mknod");must(err);defer os.RemoveAll(dir);old:=syscall.Umask(0027);defer syscall.Umask(old)
 for i,mode:=range []uint32{0600,syscall.S_IFREG|0600,syscall.S_IFIFO|0777,syscall.S_IFIFO|04777,syscall.S_IFSOCK|0600,syscall.S_IFDIR|0600,syscall.S_IFCHR|0600,syscall.S_IFBLK|0600,0xffff1600,0xffffffff}{path:=filepath.Join(dir,fmt.Sprint("mode-",i));err:=syscall.Mknod(path,mode,0);result(fmt.Sprint("mode ",i),err);if err==nil{metadata(fmt.Sprint("mode stat ",i),path)}}
 for i,dev:=range []int{0,-1,1<<32,1<<48|19,1<<63-1}{path:=filepath.Join(dir,fmt.Sprint("device-",i));err:=syscall.Mknod(path,syscall.S_IFIFO|0600,dev);result(fmt.Sprint("device ",i),err);if err==nil{metadata(fmt.Sprint("device stat ",i),path)}}
 raw:=filepath.Join(dir,string([]byte{255}));err=syscall.Mknod(raw,syscall.S_IFIFO|0600,0);result("raw name",err);if err==nil{metadata("raw stat",raw)};fifo:=filepath.Join(dir,"fifo");must(syscall.Mknod(fifo,syscall.S_IFIFO|0600,0));metadata("fifo",fifo);result("duplicate path",syscall.Mknod(fifo,syscall.S_IFIFO|0600,0))
 file:=filepath.Join(dir,"file");must(os.WriteFile(file,[]byte{11,13},0600));link:=filepath.Join(dir,"link");must(os.Symlink("file",link));dangling:=filepath.Join(dir,"dangling");must(os.Symlink("absent",dangling))
 for i,path:=range []string{file,link,dangling,filepath.Join(file,"child"),filepath.Join(dir,"absent","fifo"),"","a\x00b"}{result(fmt.Sprint("bad path ",i),syscall.Mknod(path,syscall.S_IFIFO|0600,0))};data,err:=os.ReadFile(file);must(err);fmt.Println("retained file",data)
 rd,err:=syscall.Open(fifo,syscall.O_RDONLY|syscall.O_NONBLOCK,0);must(err);wr,err:=syscall.Open(fifo,syscall.O_WRONLY|syscall.O_NONBLOCK,0);must(err);n,err:=syscall.Write(wr,[]byte{0,255,17,19});must(err);duplicate,err:=syscall.Dup(rd);must(err);must(syscall.Close(rd));data=[]byte{31,37,41,43,47,53,59,61};count,err:=syscall.Read(duplicate,data[2:6]);must(err);fmt.Println("fifo bytes",n,count,data);must(syscall.Close(wr));count,err=syscall.Read(duplicate,data[2:6]);result("fifo EOF",err);fmt.Println("EOF count",count,"retained",data);must(syscall.Close(duplicate));_,err=syscall.Read(duplicate,data[2:6]);result("closed FIFO",err);fmt.Println("closed retained",data)
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
