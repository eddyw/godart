package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLinuxCreat(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Creat requires Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/creation\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"os";"syscall")
func must(err error){if err!=nil{panic(err)}}
func main(){
 dir,err:=os.MkdirTemp("","creat");must(err);defer os.RemoveAll(dir);path:=dir+"/actual"
 fd,err:=syscall.Creat(path,0600);must(err);defer syscall.Close(fd)
 var stat syscall.Stat_t;must(syscall.Fstat(fd,&stat));fmt.Println("new",stat.Mode&0777==0600,stat.Size==0)
 n,err:=syscall.Write(fd,[]byte("actual"));must(err);fmt.Println("write",n)
 n,err=syscall.Read(fd,make([]byte,6));fmt.Println("write only",n,err==syscall.EBADF)
 duplicate,err:=syscall.Dup(fd);must(err);defer syscall.Close(duplicate);must(syscall.Close(fd));n,err=syscall.Write(duplicate,[]byte(" bytes"));must(err)
 data,err:=os.ReadFile(path);must(err);fmt.Println("owner",n,string(data)=="actual bytes")
 fd,err=syscall.Creat(path,0700);must(err);must(syscall.Fstat(fd,&stat));fmt.Println("truncate",stat.Size==0,stat.Mode&0777==0600);must(syscall.Close(fd))
 fd,err=syscall.Creat(path+"\x00suffix",0600);fmt.Println("NUL",fd,err==syscall.EINVAL)
 fd,err=syscall.Creat(dir+"/missing/file",0600);fmt.Println("missing parent",fd,err==syscall.ENOENT)
 fd,err=syscall.Creat(dir,0600);fmt.Println("directory",fd,err==syscall.EISDIR)
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
