package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestLinuxUtime(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Utime requires Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/utime\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"os";"path/filepath";"syscall";"time")
func must(err error){if err!=nil{panic(err)}}
func show(label string,err error){code:=syscall.Errno(0);if err!=nil{code=err.(syscall.Errno)};fmt.Println(label,uint64(code))}
func stat(path string)syscall.Stat_t{var s syscall.Stat_t;must(syscall.Stat(path,&s));return s}
func main(){
 base,err:=os.MkdirTemp("","utime-");must(err);defer os.RemoveAll(base)
 path:=filepath.Join(base,"raw-\xff");must(os.WriteFile(path,[]byte("value"),0600))
 link:=filepath.Join(base,"link");must(os.Symlink("raw-\xff",link))
 var linkBefore syscall.Stat_t;must(syscall.Lstat(link,&linkBefore))
 for _,seconds:=range []int64{1234567890,-1,9223372036854775807,-9223372036854775808}{
  value:=syscall.Utimbuf{Actime:seconds,Modtime:seconds};before:=value
  show("value",syscall.Utime(link,&value));s:=stat(path)
  fmt.Println("result",s.Atim,s.Mtim,"input",value==before)
 }
 value:=syscall.Utimbuf{Actime:1234567890,Modtime:1234567891};must(syscall.Utime(path,&value));before:=stat(path)
 start:=time.Now().Unix();show("nil",syscall.Utime(path,nil));after:=stat(path)
 fmt.Println("nil result",after.Atim.Sec>=start&&after.Atim.Sec<=time.Now().Unix(),after.Mtim.Sec>=start&&after.Mtim.Sec<=time.Now().Unix(),before.Atim==after.Atim,before.Mtim==after.Mtim)
 var linkAfter syscall.Stat_t;must(syscall.Lstat(link,&linkAfter));fmt.Println("follow",linkBefore.Mtim==linkAfter.Mtim)
 for _,name:=range []string{filepath.Join(base,"missing"),"","bad\x00path"}{
  show("path",syscall.Utime(name,&value));show("nil path",syscall.Utime(name,nil))
 }
 moved:=path+"-moved";must(os.Rename(path,moved));show("old name",syscall.Utime(path,&value));must(syscall.Utime(moved,&value));s:=stat(moved);fmt.Println("renamed",s.Atim,s.Mtim)
 must(os.Remove(moved));show("dangling",syscall.Utime(link,&value))
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
	for _, target := range []string{"vm", "aot"} {
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
