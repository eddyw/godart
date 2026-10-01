package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestLinuxSyscallTimestamps(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux syscall timestamps require Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/syscalltimestamps\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"os";"path/filepath";"syscall";"time")
func must(err error){if err!=nil{panic(err)}}
func show(label string,err error){code:=syscall.Errno(0);if err!=nil{code=err.(syscall.Errno)};fmt.Println(label,uint64(code))}
func stat(path string)syscall.Stat_t{var s syscall.Stat_t;must(syscall.Stat(path,&s));return s}
func main(){
 base,err:=os.MkdirTemp("","syscall-timestamps-");must(err);defer os.RemoveAll(base)
 path:=filepath.Join(base,"raw-\xff");must(os.WriteFile(path,[]byte("value"),0600))
 link:=filepath.Join(base,"link");must(os.Symlink("raw-\xff",link))
 var linkBefore syscall.Stat_t;must(syscall.Lstat(link,&linkBefore))
 fd,err:=syscall.Open(path,syscall.O_RDONLY,0);must(err);defer syscall.Close(fd)
 directory,err:=syscall.Open(base,syscall.O_RDONLY|syscall.O_DIRECTORY,0);must(err);defer syscall.Close(directory)
 moved:=base+"-moved";must(os.Rename(base,moved));defer os.RemoveAll(moved)
 path=filepath.Join(moved,"raw-\xff");link=filepath.Join(moved,"link")
 nano:=[]syscall.Timespec{{Sec:1234567890,Nsec:123456789},{Sec:1234567891,Nsec:987654321}}
 micro:=[]syscall.Timeval{{Sec:1234567892,Usec:123456},{Sec:1234567893,Usec:654321}}
 for _,length:=range []int{0,1,3}{
  show("nano length",syscall.UtimesNano(path,make([]syscall.Timespec,length)))
  show("micro length",syscall.Utimes(path,make([]syscall.Timeval,length)))
  show("relative length",syscall.Futimesat(directory,"raw-\xff",make([]syscall.Timeval,length)))
  show("fd length",syscall.Futimes(fd,make([]syscall.Timeval,length)))
 }
 must(syscall.UtimesNano(link,nano));s:=stat(path);fmt.Println("nano",s.Atim,s.Mtim)
 var linkAfter syscall.Stat_t;must(syscall.Lstat(link,&linkAfter));fmt.Println("follow",linkBefore.Mtim==linkAfter.Mtim)
 must(syscall.Utimes(path,micro));s=stat(path);fmt.Println("micro",s.Atim,s.Mtim)
 must(syscall.Futimesat(directory,"raw-\xff",micro));s=stat(path);fmt.Println("relative",s.Atim,s.Mtim)
 must(syscall.Futimes(fd,micro));s=stat(path);fmt.Println("fd",s.Atim,s.Mtim)
 for _,usec:=range []int64{-1,1000000,9223372036854775807}{
  value:=[]syscall.Timeval{{Sec:1234567890,Usec:usec},{Sec:1234567890,Usec:usec}}
  show("micro value",syscall.Utimes(path,value));s=stat(path);fmt.Println("micro result",s.Atim,s.Mtim)
  show("relative value",syscall.Futimesat(directory,"raw-\xff",value));s=stat(path);fmt.Println("relative result",s.Atim,s.Mtim)
 }
 for _,nsec:=range []int64{-1,1000000000}{
  show("nano value",syscall.UtimesNano(path,[]syscall.Timespec{{Sec:1234567890,Nsec:nsec},nano[1]}))
 }
 must(syscall.UtimesNano(path,nano));before:=stat(path)
 omitted:=[]syscall.Timespec{{Sec:-1,Nsec:1073741822},{Sec:9223372036854775807,Nsec:1073741822}}
 must(syscall.UtimesNano(path,omitted));after:=stat(path);fmt.Println("omit",before.Atim==after.Atim,before.Mtim==after.Mtim)
 show("omit absent",syscall.UtimesNano(filepath.Join(moved,"missing"),omitted))
 show("omit NUL",syscall.UtimesNano("bad\x00path",omitted))
 must(syscall.UtimesNano(path,[]syscall.Timespec{omitted[0],{Sec:1234567894,Nsec:321}}));after=stat(path);fmt.Println("mixed omit",before.Atim==after.Atim,after.Mtim)
 show("nano invalid missing",syscall.UtimesNano(filepath.Join(moved,"missing"),[]syscall.Timespec{{Nsec:-1},nano[1]}))
 badMicro:=[]syscall.Timeval{{Sec:1234567890,Usec:-1},micro[1]}
 show("micro invalid missing",syscall.Utimes(filepath.Join(moved,"missing"),badMicro))
 show("relative invalid missing",syscall.Futimesat(-1,"missing",badMicro))
 start:=time.Now().Unix();must(syscall.UtimesNano(path,[]syscall.Timespec{{Nsec:1073741823},{Nsec:1073741823}}));after=stat(path)
 fmt.Println("now",after.Atim.Sec>=start&&after.Atim.Sec<=time.Now().Unix(),after.Mtim.Sec>=start&&after.Mtim.Sec<=time.Now().Unix())
 show("nano NUL",syscall.UtimesNano("bad\x00path",nano));show("micro NUL",syscall.Utimes("bad\x00path",micro))
 show("relative NUL",syscall.Futimesat(-1,"bad\x00path",micro));show("bad directory",syscall.Futimesat(-1,"file",micro))
 show("regular directory",syscall.Futimesat(fd,"file",micro));show("bad fd",syscall.Futimes(-1,micro))
 must(os.Remove(path));show("unlinked fd",syscall.Futimes(fd,micro));var retained syscall.Stat_t;must(syscall.Fstat(fd,&retained));fmt.Println("retained",retained.Atim,retained.Mtim)
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
