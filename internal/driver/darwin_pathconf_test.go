package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestDarwinPathconf(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Pathconf requires macOS")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/pathconf\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"os";"path/filepath";"syscall")
func main(){
 if len(os.Args)>1&&os.Args[1]=="unsupported"{
  v,e:=syscall.Pathconf(".",4);if v!=0||!errors.Is(e,errors.ErrUnsupported){panic("path query must be unsupported")}
  v,e=syscall.Fpathconf(-1,4);if v!=0||!errors.Is(e,errors.ErrUnsupported){panic("descriptor query must be unsupported")}
  fmt.Println("pathconf unsupported");return
 }
 v,e:=syscall.Fpathconf(-1,4);fmt.Println("first invalid",v,e)
 dir,e:=os.MkdirTemp("","pathconf-");if e!=nil{panic(e)};defer os.RemoveAll(dir)
 path:=filepath.Join(dir,"file");if e:=os.WriteFile(path,nil,0600);e!=nil{panic(e)}
 if e:=os.Symlink("file",filepath.Join(dir,"link"));e!=nil{panic(e)}
 fd,e:=syscall.Open(path,syscall.O_RDONLY,0);if e!=nil{panic(e)}
 for _,name:=range []int{1,4,5,6,7,8,10,11,12,13,14,15,16,17,18,19,20,2,-1,999,(1<<32)|4}{
  v,e:=syscall.Pathconf(path,name);w,x:=syscall.Fpathconf(fd,name);fmt.Println("query",name,v,e,w,x)
 }
 for i,path:=range []string{dir,path,filepath.Join(dir,"link"),filepath.Join(dir,"missing"),filepath.Join(path,"child"),"","a\x00b",string([]byte{0xff})}{v,e:=syscall.Pathconf(path,4);fmt.Println("path",i,v,e)}
 for _,desc:=range []int{fd,fd+(1<<32),-1,1<<30}{v,e:=syscall.Fpathconf(desc,4);fmt.Println("fd",desc==fd,v,e)}
 if e:=os.Remove(path);e!=nil{panic(e)}
 v,e=syscall.Fpathconf(fd,4);fmt.Println("removed",v,e)
 if e:=syscall.Close(fd);e!=nil{panic(e)}
 v,e=syscall.Fpathconf(fd,4);fmt.Println("closed",v,e)
}
`)
	want := programOutcome(t, dir, "go", "run", ".")
	if want.Exit != 0 {
		t.Fatalf("native Go failed: %+v", want)
	}
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	targets := []string{"vm", "aot"}
	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			var got programResult
			if target == "vm" {
				got = programOutcome(t, dir, "dart", filepath.Join(out, "bin/main.dart"))
			} else {
				got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
			}
			if got != want {
				t.Fatalf("Go:\n%s\nGart (exit %d):\n%s", want.Text, got.Exit, got.Text)
			}
		})
	}
	for _, target := range []string{"js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			got := runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{"unsupported"})
			if got.Exit != 0 || got.Text != "pathconf unsupported" {
				t.Fatalf("unsupported target differs: %+v", got)
			}
		})
	}

}
