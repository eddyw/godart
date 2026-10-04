package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLinuxMknod(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Mknod requires Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/accept\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"os";"path/filepath";"syscall")
func must(err error){if err!=nil{panic(err)}}
func result(label string,err error){if err==nil{fmt.Println(label,0);return};code,ok:=err.(syscall.Errno);if !ok{panic("non-errno")};fmt.Println(label,int(code))}
func metadata(label,path string){var value syscall.Stat_t;must(syscall.Stat(path,&value));fmt.Printf("%s %o %d\n",label,value.Mode,value.Rdev)}
func main(){
 dir,err:=os.MkdirTemp("","mknod");must(err);defer os.RemoveAll(dir);old:=syscall.Umask(0027);defer syscall.Umask(old)
 parentPath:=filepath.Join(dir,"parent");must(os.Mkdir(parentPath,0750));parent,err:=syscall.Open(parentPath,syscall.O_RDONLY|syscall.O_DIRECTORY,0);must(err);duplicate,err:=syscall.Dup(parent);must(err);moved:=filepath.Join(dir,"moved");must(os.Rename(parentPath,moved))
 must(syscall.Mknodat(parent,"fifo",syscall.S_IFIFO|0777,0));metadata("relative fifo",filepath.Join(moved,"fifo"));result("duplicate path",syscall.Mknodat(parent,"fifo",syscall.S_IFIFO|0600,0));must(syscall.Close(parent));result("closed parent",syscall.Mknodat(parent,"closed",syscall.S_IFIFO|0600,0));must(syscall.Mknodat(duplicate,string([]byte{255}),syscall.S_IFIFO|0600,0));metadata("raw name",filepath.Join(moved,string([]byte{255})))
 for i,dev:=range []int{0,-1,1<<32,1<<48|19,1<<63-1}{name:=fmt.Sprint("device-",i);result(fmt.Sprint("device ",i),syscall.Mknodat(duplicate,name,syscall.S_IFIFO|0600,dev));metadata(fmt.Sprint("device mode ",i),filepath.Join(moved,name))}
 for i,mode:=range []uint32{0600,syscall.S_IFREG|0600,syscall.S_IFIFO|04777,syscall.S_IFSOCK|0600,syscall.S_IFDIR|0600,syscall.S_IFCHR|0600,syscall.S_IFBLK|0600,0xffffffff}{path:=filepath.Join(dir,fmt.Sprint("mode-",i));err:=syscall.Mknod(path,mode,0);result(fmt.Sprint("mode ",i),err);if err==nil{metadata(fmt.Sprint("mode stat ",i),path)}}
 absolute:=filepath.Join(dir,"absolute");must(syscall.Mknodat(-1,absolute,syscall.S_IFIFO|0600,0));metadata("absolute",absolute)
 result("invalid parent",syscall.Mknodat(-1,"relative",syscall.S_IFIFO|0600,0));result("NUL before parent",syscall.Mknodat(-1,"a\x00b",syscall.S_IFIFO|0600,0));result("NUL",syscall.Mknod("a\x00b",syscall.S_IFIFO|0600,0));result("empty",syscall.Mknodat(duplicate,"",syscall.S_IFIFO|0600,0));result("missing",syscall.Mknodat(duplicate,"missing/fifo",syscall.S_IFIFO|0600,0))
 regular,err:=syscall.Open(filepath.Join(dir,"mode-0"),syscall.O_RDONLY,0);must(err);result("regular parent",syscall.Mknodat(regular,"fifo",syscall.S_IFIFO|0600,0));must(syscall.Close(regular));must(syscall.Close(duplicate))
 fd,err:=syscall.Open(filepath.Join(moved,"fifo"),syscall.O_RDWR|syscall.O_NONBLOCK,0);must(err);n,err:=syscall.Write(fd,[]byte{0,255,17,19});must(err);duplicate,err=syscall.Dup(fd);must(err);must(syscall.Close(fd));data:=[]byte{31,37,41,43,47,53,59,61};count,err:=syscall.Read(duplicate,data[2:6]);must(err);fmt.Println("fifo bytes",n,count,data);must(syscall.Close(duplicate));result("closed FIFO",func()error{_,err:=syscall.Write(duplicate,[]byte{23});return err}())
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
