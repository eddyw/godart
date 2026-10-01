package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestDarwinChflags(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Chflags requires macOS")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/chflags\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"os";"path/filepath";"syscall")
func main(){
 if len(os.Args)>1&&os.Args[1]=="unsupported"{
  if !errors.Is(syscall.Chflags("missing",1),errors.ErrUnsupported){panic("path mutation must be unsupported")}
  if !errors.Is(syscall.Fchflags(-1,1),errors.ErrUnsupported){panic("descriptor mutation must be unsupported")}
  fmt.Println("chflags unsupported");return
 }
 fmt.Println("first invalid",syscall.Fchflags(-1,1))
 dir,e:=os.MkdirTemp("","chflags-");if e!=nil{panic(e)};defer os.RemoveAll(dir)
 path:=filepath.Join(dir,"file");if e:=os.WriteFile(path,nil,0600);e!=nil{panic(e)}
 link:=filepath.Join(dir,"link");if e:=os.Symlink("file",link);e!=nil{panic(e)}
 fd,e:=syscall.Open(path,syscall.O_RDONLY,0);if e!=nil{panic(e)};defer syscall.Close(fd)
 show:=func(label string){var st syscall.Stat_t;if e:=syscall.Fstat(fd,&st);e!=nil{panic(e)};fmt.Println(label,st.Flags)}
 for _,flags:=range []int{1,0,(1<<32)|1,0}{
  fmt.Println("path",flags,syscall.Chflags(path,flags));show("flags")
  fmt.Println("fd",flags,syscall.Fchflags(fd,flags));show("flags")
 }
 fmt.Println("link",syscall.Chflags(link,1));show("link flags")
 fmt.Println("wrapped fd",syscall.Fchflags(fd+(1<<32),0));show("reset flags")
 for i,path:=range []string{filepath.Join(dir,"missing"),filepath.Join(path,"child"),"","a\x00b",string([]byte{0xff})}{fmt.Println("bad path",i,syscall.Chflags(path,1));show("unchanged")}
 if e:=os.Remove(path);e!=nil{panic(e)}
 fmt.Println("removed",syscall.Fchflags(fd,1));show("removed flags")
 fmt.Println("reset removed",syscall.Fchflags(fd,0));show("removed reset")
 if e:=syscall.Close(fd);e!=nil{panic(e)}
 fmt.Println("closed",syscall.Fchflags(fd,1))
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
			if got.Exit != 0 || got.Text != "chflags unsupported" {
				t.Fatalf("unsupported target differs: %+v", got)
			}
		})
	}

}
