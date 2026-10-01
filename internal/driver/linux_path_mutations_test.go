package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestLinuxPathMutations(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("descriptor-relative path syscalls require Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/pathmutations\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"os";"path/filepath";"syscall")
func must(err error){if err!=nil{panic(err)}}
func show(label string,err error){
 code:=syscall.Errno(0)
 if err!=nil{var ok bool;code,ok=err.(syscall.Errno);if !ok{panic(err)}}
 fmt.Println(label,uint64(code))
}
func metadata(label,path string){
 var st syscall.Stat_t;err:=syscall.Lstat(path,&st);show(label,err)
 if err==nil{fmt.Println(st.Mode&syscall.S_IFMT,st.Mode&07777,st.Uid,st.Gid)}
}
func main(){
 base,err:=os.MkdirTemp("","path-mutations-");must(err);defer os.RemoveAll(base)
 left:=filepath.Join(base,"left");right:=filepath.Join(base,"right")
 must(os.Mkdir(left,0700));must(os.Mkdir(right,0700))
 from,err:=syscall.Open(left,syscall.O_RDONLY|syscall.O_DIRECTORY,0);must(err);defer syscall.Close(from)
 to,err:=syscall.Open(right,syscall.O_RDONLY|syscall.O_DIRECTORY,0);must(err);defer syscall.Close(to)
 must(os.Rename(left,left+"-moved"));left+="-moved"
 must(os.Rename(right,right+"-moved"));right+="-moved"
 file:=filepath.Join(left,"file");must(os.WriteFile(file,[]byte("value"),0600))
 must(os.Mkdir(filepath.Join(left,"sub"),0700))
 must(os.Symlink("sub",filepath.Join(left,"linkdir")))
 must(os.Symlink("file",filepath.Join(left,"link")))
 must(os.Symlink("absent",filepath.Join(left,"dangling")))
 raw:=string([]byte{'r',0xff,'w'});must(os.WriteFile(filepath.Join(left,raw),[]byte("raw"),0600))
 must(os.WriteFile(filepath.Join(left,"r\uFFFDw"),[]byte("decoy"),0600))
 oldMask:=syscall.Umask(0027)
 for i,mode:=range []uint32{0,0700,0777,04777,02777,01777,0xffffffff}{
  name:=fmt.Sprint("mode",i);show("mkdir mode",syscall.Mkdirat(from,name,mode));metadata("mode",filepath.Join(left,name))
 }
 syscall.Umask(oldMask)
 for _,path:=range []string{"mode0","file","file/child","absent/child","linkdir/child","../parent-child","dangling/child","","bad\x00name",raw+"-dir"}{
  show("mkdir path",syscall.Mkdirat(from,path,0700))
 }
 show("mkdir invalid descriptor",syscall.Mkdirat(-1,"new",0700))
 show("mkdir invalid descriptor nul",syscall.Mkdirat(-1,"bad\x00name",0700))
 show("mkdir absolute",syscall.Mkdirat(-1,filepath.Join(right,"absolute"),0700))
 for _,ids:=range [][2]int{{-1,-1},{os.Geteuid(),os.Getegid()},{-2,-2},{1<<32+os.Geteuid(),1<<32+os.Getegid()}}{
  show("chown IDs",syscall.Fchownat(from,"file",ids[0],ids[1],0));metadata("owner",file)
 }
 for _,item:=range []struct{path string;flags int}{
  {"file",0},{"link",0},{"link",0x100},{"dangling",0},{"dangling",0x100},
  {raw,0},{"absent",0},{"file/child",0},{"",0},{"bad\x00name",0},
  {"file",1},{"absent",1},{"bad\x00name",1},{"file",-1},
 }{show("chown path",syscall.Fchownat(from,item.path,-1,-1,item.flags))}
 metadata("link owner",filepath.Join(left,"link"))
 filefd,err:=syscall.Open(file,syscall.O_RDONLY,0);must(err)
 show("chown empty descriptor",syscall.Fchownat(filefd,"",-1,-1,0x1000));must(syscall.Close(filefd))
 show("chown invalid descriptor",syscall.Fchownat(-1,"file",-1,-1,0))
 show("chown invalid descriptor flags",syscall.Fchownat(-1,"file",-1,-1,1))
 show("chown absolute",syscall.Fchownat(-1,file,-1,-1,0))
 show("rename",syscall.Renameat(from,"file",to,"renamed"))
 data,err:=os.ReadFile(filepath.Join(right,"renamed"));must(err);fmt.Println("renamed value",string(data))
 must(os.WriteFile(filepath.Join(left,"replace"),[]byte("replacement"),0600))
 show("rename replace",syscall.Renameat(from,"replace",to,"renamed"))
 data,err=os.ReadFile(filepath.Join(right,"renamed"));must(err);fmt.Println("replacement value",string(data))
 show("rename directory",syscall.Renameat(from,"sub",to,"directory"))
 metadata("moved child",filepath.Join(right,"directory/child"))
 show("rename raw",syscall.Renameat(from,raw,to,raw))
 data,err=os.ReadFile(filepath.Join(right,raw));must(err);fmt.Println("raw value",string(data))
 data,err=os.ReadFile(filepath.Join(left,"r\uFFFDw"));must(err);fmt.Println("raw decoy",string(data))
 show("rename symlink",syscall.Renameat(from,"link",to,"link"));metadata("moved link",filepath.Join(right,"link"))
 must(os.WriteFile(filepath.Join(left,"source"),[]byte("keep"),0600))
 for _,pair:=range [][2]string{{"absent","unused"},{"source","absent/child"},{"source","directory"},{"mode0","renamed"},{"","unused"},{"source",""},{"bad\x00name","unused"},{"source","bad\x00name"}}{
  show("rename error",syscall.Renameat(from,pair[0],to,pair[1]))
 }
 show("rename source descriptor",syscall.Renameat(-1,"source",to,"unused"))
 show("rename target descriptor",syscall.Renameat(from,"source",-1,"unused"))
 show("rename nul precedence",syscall.Renameat(-1,"source",-1,"bad\x00name"))
 show("rename absolute",syscall.Renameat(-1,filepath.Join(left,"source"),-1,filepath.Join(right,"absolute-source")))
 must(os.Symlink("absolute-source",filepath.Join(right,"retained-link")))
 for _,path:=range []string{"link","retained-link","directory","renamed",raw,"absent","absolute-source/child","","bad\x00name"}{
  show("unlink path",syscall.Unlinkat(to,path))
 }
 data,err=os.ReadFile(filepath.Join(right,"absolute-source"));must(err);fmt.Println("retained target",string(data))
 show("unlink invalid descriptor",syscall.Unlinkat(-1,"absolute-source"))
 show("unlink invalid descriptor nul",syscall.Unlinkat(-1,"bad\x00name"))
 show("unlink absolute",syscall.Unlinkat(-1,filepath.Join(right,"absolute-source")))
 closed,err:=syscall.Dup(from);must(err);must(syscall.Close(closed))
 show("mkdir closed descriptor",syscall.Mkdirat(closed,"new",0700))
 show("chown closed descriptor",syscall.Fchownat(closed,"dangling",-1,-1,0x100))
 show("unlink closed descriptor",syscall.Unlinkat(closed,"dangling"))
 show("rename closed descriptor",syscall.Renameat(closed,"dangling",to,"unused"))
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
