package driver_test

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestDarwinStatfs(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Statfs requires macOS")
	}
	testStatfs(t)
}

func TestLinuxStatfs(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Statfs requires Linux")
	}
	testStatfs(t)
}

func testStatfs(t *testing.T) {
	t.Helper()
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/statfs\n\ngo 1.27\n")
	source := `package main
import("errors";"fmt";"os";"path/filepath";"syscall")
func must(err error){if err!=nil{panic(err)}}
func stable(stat syscall.Statfs_t)syscall.Statfs_t{stat.Bfree,stat.Bavail,stat.Files,stat.Ffree=0,0,0,0;return stat}
func main(){
 marker:=syscall.Statfs_t{Bsize:17,Reserved:[8]uint32{1,2,3}}
 if len(os.Args)>1&&os.Args[1]=="unsupported"{
  stat:=marker
  for _,err:=range []error{syscall.Statfs("/",&stat),syscall.Statfs("/",nil),syscall.Fstatfs(-1,&stat),syscall.Fstatfs(-1,nil)}{
   if !errors.Is(err,errors.ErrUnsupported)||stat!=marker{panic("unsupported call changed output")}
  }
  fmt.Println("statfs unsupported");return
 }
 dir,err:=os.MkdirTemp("","statfs-");must(err);defer os.RemoveAll(dir)
 path:=filepath.Join(dir,"file");must(os.WriteFile(path,[]byte("x"),0600))
 file,err:=os.Open(path);must(err);defer file.Close()
 directory,err:=os.Open(dir);must(err);defer directory.Close()
 link:=filepath.Join(dir,"link");must(os.Symlink(path,link))
 dangling:=filepath.Join(dir,"dangling");must(os.Symlink(filepath.Join(dir,"absent"),dangling))
 for i,p:=range []string{dir,path,link,dangling,filepath.Join(dir,"missing"),path+"/child","","a\x00b",filepath.Join(dir,string([]byte{0xff}))}{
  stat:=marker;err:=syscall.Statfs(p,&stat)
  fmt.Println("path",i,err,stat==marker)
  fmt.Println("nil",i,syscall.Statfs(p,nil))
  if err==nil{fmt.Printf("path data %+v\n",stable(stat));fmt.Println("space",stat.Blocks>0,stat.Bfree<=stat.Blocks,stat.Bavail<=stat.Bfree)}
 }
 for i,fd:=range []int{int(file.Fd()),int(directory.Fd()),-1,1<<30,int(file.Fd())+1<<32}{
  stat:=marker;err:=syscall.Fstatfs(fd,&stat)
  fmt.Println("fd",i,err,stat==marker)
  fmt.Println("fdnil",i,syscall.Fstatfs(fd,nil))
  if err==nil{fmt.Printf("fd data %+v\n",stable(stat));fmt.Println("space",stat.Blocks>0,stat.Bfree<=stat.Blocks,stat.Bavail<=stat.Bfree)}
 }
 must(os.Remove(path))
 stat:=marker
 fmt.Println("retained",syscall.Fstatfs(int(file.Fd()),&stat),stat==marker)
 fmt.Println("removed",syscall.Statfs(path,&stat))
 fd:=int(file.Fd());must(file.Close());stat=marker
 fmt.Println("closed",syscall.Fstatfs(fd,&stat),stat==marker)
}
`
	if runtime.GOOS == "linux" {
		source = strings.Replace(source, "Reserved:[8]uint32{1,2,3}", "Spare:[4]int64{1,2,3}", 1)
	}
	writeFile(t, filepath.Join(dir, "main.go"), source)
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
			if got.Exit != 0 || got.Text != "statfs unsupported" {
				t.Fatalf("unsupported target differs: %+v", got)
			}
		})
	}

}
