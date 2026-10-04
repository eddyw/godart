package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLinuxInotify(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Inotify requires Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/accept\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("encoding/binary";"fmt";"os";"strconv";"strings";"syscall")
func must(err error){if err!=nil{panic(err)}}
type event struct{wd int32;mask,cookie uint32;name string}
func receive(fd int)[]event{
 buffer:=make([]byte,16384);for i:=range buffer{buffer[i]=0xa5};n,err:=syscall.Read(fd,buffer[7:len(buffer)-7]);must(err)
 for _,b:=range buffer[:7]{if b!=0xa5{panic("prefix")}};for _,b:=range buffer[len(buffer)-7:]{if b!=0xa5{panic("suffix")}}
 data:=buffer[7:7+n];var events []event
 for len(data)>0{if len(data)<syscall.SizeofInotifyEvent{panic("short event")};length:=int(binary.NativeEndian.Uint32(data[12:]));if length>len(data)-syscall.SizeofInotifyEvent{panic("short name")};events=append(events,event{int32(binary.NativeEndian.Uint32(data)),binary.NativeEndian.Uint32(data[4:]),binary.NativeEndian.Uint32(data[8:]),strings.TrimRight(string(data[16:16+length]),"\x00")});data=data[16+length:]}
 return events
}
func main(){
 plain,err:=syscall.InotifyInit();must(err);must(syscall.Close(plain))
 for _,flags:=range []int{0,syscall.IN_NONBLOCK,syscall.IN_CLOEXEC,syscall.IN_NONBLOCK|syscall.IN_CLOEXEC}{fd,err:=syscall.InotifyInit1(flags);must(err);data,err:=os.ReadFile(fmt.Sprintf("/proc/self/fdinfo/%d",fd));must(err);found:=false;for _,line:=range strings.Split(string(data),"\n"){if strings.HasPrefix(line,"flags:"){bits,err:=strconv.ParseUint(strings.TrimSpace(strings.TrimPrefix(line,"flags:")),8,64);must(err);fmt.Println("flags",bits&syscall.O_NONBLOCK!=0,bits&syscall.O_CLOEXEC!=0);found=true}};if !found{panic("fdinfo flags")};must(syscall.Close(fd))}
 bad,err:=syscall.InotifyInit1(1<<30);fmt.Println("invalid flags",bad,err==syscall.EINVAL)
 fd,err:=syscall.InotifyInit1(syscall.IN_NONBLOCK|syscall.IN_CLOEXEC);must(err);defer syscall.Close(fd)
 dir,err:=os.MkdirTemp("","watch");must(err);defer os.RemoveAll(dir);path:=dir+"/raw\xff";must(syscall.Mkdir(path,0700))
 mask:=uint32(syscall.IN_CREATE|syscall.IN_MODIFY|syscall.IN_CLOSE_WRITE|syscall.IN_MOVED_FROM|syscall.IN_MOVED_TO|syscall.IN_DELETE)
 wd,err:=syscall.InotifyAddWatch(fd,path,mask);must(err);again,err:=syscall.InotifyAddWatch(fd,path,mask);must(err);fmt.Println("actual watch",wd>=0,wd==again)
 added,err:=syscall.InotifyAddWatch(fd,path,syscall.IN_ATTRIB|syscall.IN_MASK_ADD);must(err);fmt.Println("mask add",wd==added)
 buffer:=make([]byte,128);n,err:=syscall.Read(fd,buffer);fmt.Println("empty",n,err==syscall.EAGAIN)
 source,err:=syscall.Open(path+"/sample",syscall.O_WRONLY|syscall.O_CREAT|syscall.O_EXCL,0600);must(err);w,err:=syscall.Write(source,[]byte("actual"));must(err);if w!=6{panic("write")};must(syscall.Close(source));must(syscall.Chmod(path+"/sample",0640));must(syscall.Rename(path+"/sample",path+"/renamed"));must(syscall.Unlink(path+"/renamed"))
 var create,modify,closeWrite,attribute,movedFrom,movedTo,deleted bool;var cookie uint32
 for _,e:=range receive(fd){if e.wd!=int32(wd){panic("watch ID")};if e.mask&syscall.IN_CREATE!=0&&e.name=="sample"{create=true};if e.mask&syscall.IN_MODIFY!=0&&e.name=="sample"{modify=true};if e.mask&syscall.IN_CLOSE_WRITE!=0&&e.name=="sample"{closeWrite=true};if e.mask&syscall.IN_ATTRIB!=0&&e.name=="sample"{attribute=true};if e.mask&syscall.IN_MOVED_FROM!=0&&e.name=="sample"{movedFrom=true;cookie=e.cookie};if e.mask&syscall.IN_MOVED_TO!=0&&e.name=="renamed"{movedTo=e.cookie!=0&&e.cookie==cookie};if e.mask&syscall.IN_DELETE!=0&&e.name=="renamed"{deleted=true}}
 fmt.Println("events",create,modify,closeWrite,attribute,movedFrom,movedTo,deleted)
 duplicate,err:=syscall.Dup(fd);must(err);defer syscall.Close(duplicate);must(syscall.Close(fd))
 success,err:=syscall.InotifyRmWatch(duplicate,uint32(wd));must(err);ignored:=false;for _,e:=range receive(duplicate){if e.wd==int32(wd)&&e.mask&syscall.IN_IGNORED!=0&&e.name==""{ignored=true}};fmt.Println("remove",success,ignored)
 success,err=syscall.InotifyRmWatch(duplicate,uint32(wd));fmt.Println("removed",success,err==syscall.EINVAL)
 wd,err=syscall.InotifyAddWatch(duplicate,path,mask);must(err);fmt.Println("duplicate lifetime",wd>=0)
 wd,err=syscall.InotifyAddWatch(duplicate,path+"/missing",mask);fmt.Println("missing",wd,err==syscall.ENOENT)
 wd,err=syscall.InotifyAddWatch(-1,path,mask);fmt.Println("bad descriptor",wd,err==syscall.EBADF)
 wd,err=syscall.InotifyAddWatch(-1,path+"\x00suffix",mask);fmt.Println("NUL before descriptor",wd,err==syscall.EINVAL)
 wd,err=syscall.InotifyAddWatch(duplicate,"",mask);fmt.Println("empty path",wd,err==syscall.ENOENT)
 success,err=syscall.InotifyRmWatch(fd,1);fmt.Println("closed remove",success,err==syscall.EBADF)
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
