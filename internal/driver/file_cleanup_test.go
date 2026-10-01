package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestFileResourceCleanup(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("file cleanup requires a native filesystem")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/filecleanup\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"os";"runtime";"strconv";"syscall";"time")
func must(err error){if err!=nil{panic(err)}}
func count(path string)int{
 var target syscall.Stat_t;must(syscall.Stat(path,&target))
 entries,err:=os.ReadDir("/dev/fd");must(err)
 count:=0
 for _,entry:=range entries{
  fd,err:=strconv.Atoi(entry.Name());if err!=nil{continue}
  var stat syscall.Stat_t
  if syscall.Fstat(fd,&stat)==nil&&stat.Dev==target.Dev&&stat.Ino==target.Ino{count++}
 }
 return count
}
func abandon(open func()(*os.File,error)){f,err:=open();must(err);runtime.KeepAlive(f)}
func main(){
 base,err:=os.MkdirTemp("","file-cleanup");must(err);defer os.RemoveAll(base)
 must(os.WriteFile(base+"/file",[]byte("contents"),0600));must(os.Mkdir(base+"/sub",0700))
 root,err:=os.OpenRoot(base);must(err);defer root.Close()
 for _,kind:=range []string{"ordinary","rooted","adopted"}{
  for _,name:=range []string{"file","sub"}{
   path:=base+"/"+name
   open:=func()(*os.File,error){
    if kind=="rooted"{return root.Open(name)}
    if kind=="adopted"{fd,err:=syscall.Open(path,syscall.O_RDONLY,0);if err!=nil{return nil,err};return os.NewFile(uintptr(fd),path),nil}
    return os.Open(path)
   }
   live,err:=open();must(err);copy:=*live
   runtime.GC();time.Sleep(20*time.Millisecond)
   fmt.Println("live",kind,name,count(path)==1)
   _,err=live.Stat();must(err);must(copy.Close());fmt.Println("closed",count(path)==0)
   replacement,err:=open();must(err)
   runtime.GC();time.Sleep(20*time.Millisecond)
   _,err=replacement.Stat();must(err);fmt.Println("replacement",count(path)==1);must(replacement.Close())
   for i:=0;i<64;i++{abandon(open)}
   deadline:=time.Now().Add(4*time.Second)
   for count(path)!=0{
    runtime.GC();time.Sleep(10*time.Millisecond)
    if time.Now().After(deadline){panic("file descriptors remain open")}
   }
   fmt.Println("collected",count(path)==0)
  }
 }
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
