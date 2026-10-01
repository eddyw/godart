package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestNativeAccess(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native access requires a native filesystem")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/nativeaccess\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"os";"path/filepath";"syscall")
func must(err error) { if err != nil { panic(err) } }
func show(label,path string, mode uint32) {
 err:=syscall.Access(path,mode)
 if err==nil { fmt.Println(label,mode,0);return }
 code,ok:=err.(syscall.Errno)
 fmt.Println(label,mode,uint64(code),ok)
}
func main() {
 dir,err:=os.MkdirTemp("","native-access-");must(err);defer os.RemoveAll(dir)
 file:=filepath.Join(dir,"file")
 must(os.WriteFile(file,[]byte("data"),0600))
 raw:=filepath.Join(dir,string([]byte{'r',0xff,'w'}))
 err=os.WriteFile(raw,nil,0600)
 var rawCode syscall.Errno
 errors.As(err,&rawCode)
 fmt.Println("raw create",uint64(rawCode))
 link:=filepath.Join(dir,"link");must(os.Symlink(file,link))
 dangling:=filepath.Join(dir,"dangling");must(os.Symlink(filepath.Join(dir,"absent"),dangling))
 for _,permission:=range []os.FileMode{0000,0400,0200,0100,0600,0700} {
  must(os.Chmod(file,permission))
  fmt.Printf("permission %o\n",permission)
  for _,mode:=range []uint32{0,1,2,4,3,5,6,7,8,0xffffffff} {
   show("file",file,mode)
   show("symlink",link,mode)
  }
 }
 for _,item:=range []struct{name,path string}{
  {"directory",dir},{"raw",raw},{"empty",""},{"absent",filepath.Join(dir,"absent")},
  {"dangling",dangling},{"not-directory",file+"/child"},{"nul",file+"\x00tail"},
 } {
  for _,mode:=range []uint32{0,1,2,4,8,0xffffffff} {show(item.name,item.path,mode)}
 }
 must(os.Chmod(file,0600))
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
}
