package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestRootDescriptorCleanup(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("root-open errors require a native filesystem")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/rootcleanup\n\ngo 1.27\n")
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
func abandon(path string){root,err:=os.OpenRoot(path);must(err);runtime.KeepAlive(root)}
func main(){
 dir,err:=os.MkdirTemp("","root-cleanup");must(err);defer os.RemoveAll(dir)
 live,err:=os.OpenRoot(dir);must(err)
 runtime.GC();time.Sleep(20*time.Millisecond)
 fmt.Println("live",count(dir)==1)
 _,err=live.Stat(".");must(err);must(live.Close());fmt.Println("closed",count(dir)==0)
 for i:=0;i<64;i++{abandon(dir)}
 deadline:=time.Now().Add(4*time.Second)
 for count(dir)!=0{
  runtime.GC();time.Sleep(10*time.Millisecond)
  if time.Now().After(deadline){panic("root descriptors remain open")}
 }
 fmt.Println("collected",count(dir)==0)
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
