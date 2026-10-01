package driver_test

import (
	"go/build"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/parity"
)

func TestLinuxFaccessat(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Faccessat requires Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/faccessat\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"os";"path/filepath";"syscall")
func must(err error){if err!=nil{panic(err)}}
func main(){
 parent,err:=os.MkdirTemp("","faccessat-");must(err);defer os.RemoveAll(parent)
 root:=filepath.Join(parent,"root");must(os.Mkdir(root,0700))
 file:=filepath.Join(root,"file");must(os.WriteFile(file,nil,0600))
 raw:=string([]byte{'r',0xff,'w'});must(os.WriteFile(filepath.Join(root,raw),nil,0600))
 must(os.Mkdir(filepath.Join(root,"sub"),0700))
 must(os.Symlink("file",filepath.Join(root,"link")))
 must(os.Symlink("absent",filepath.Join(root,"dangling")))
 fd,err:=syscall.Open(root,syscall.O_RDONLY|syscall.O_DIRECTORY,0);must(err);defer syscall.Close(fd)
 filefd,err:=syscall.Open(file,syscall.O_RDONLY,0);must(err);defer syscall.Close(filefd)
 moved:=filepath.Join(parent,"moved");must(os.Rename(root,moved));file=filepath.Join(moved,"file")
 show:=func(label string,base int,path string,mode uint32,flags int){
  err:=syscall.Faccessat(base,path,mode,flags)
  code:=syscall.Errno(0)
  if err!=nil{var ok bool;code,ok=err.(syscall.Errno);if !ok{panic(err)}}
  fmt.Println(label,uint64(code))
 }
 for _,perm:=range []os.FileMode{0,0400,0200,0100,0600,0700,0001}{
  must(os.Chmod(file,perm))
  for _,mode:=range []uint32{0,1,2,3,4,5,6,7,8,0xffffffff}{
   for _,flags:=range []int{0,0x100,0x200,0x300}{show("mode",fd,"file",mode,flags)}
  }
 }
 for _,item:=range []struct{label,path string;flags int}{
  {"dot","./file",0},{"parent","sub/../file",0},{"follow","link",0},
  {"nofollow file","file",0x100},{"nofollow link","link",0x100},
  {"dangling","dangling",0},{"dangling nofollow","dangling",0x100},
  {"absent","absent",0},{"not-directory","file/child",0},
  {"raw",raw,0},{"empty","",0},{"nul","file\x00tail",0},
  {"invalid flags","file",1},{"invalid all flags","file",-1},
  {"invalid absent","absent",1},{"invalid nul","file\x00tail",1},
 }{show(item.label,fd,item.path,4,item.flags)}
 show("empty file descriptor",filefd,"",0,0x1000)
 show("empty effective",filefd,"",4,0x1200)
 show("invalid descriptor",-1,"file",4,0)
 show("absolute ignores descriptor",-1,file,0,0)
 show("absolute nofollow",-1,file,0,0x100)
 show("invalid descriptor flags",-1,"file",4,1)
 show("invalid descriptor nul",-1,"file\x00tail",4,0)
 closed,err:=syscall.Dup(fd);must(err);must(syscall.Close(closed))
 show("closed descriptor",closed,"file",4,0)
 for _,path:=range []string{"/", "/etc/passwd"}{
  for _,mode:=range []uint32{0,1,2,4,7}{
   for _,flags:=range []int{0,0x100,0x200,0x300}{show("system",fd,path,mode,flags)}
  }
 }
 if fixtures:=os.Getenv("GART_FACCESSAT_FIXTURES");fixtures!=""{
  if os.Getuid()!=65534 || os.Geteuid()!=65534 || os.Getgid()!=65534 {panic("credential fixture IDs")}
  groups,err:=os.Getgroups();must(err);if len(groups)!=1 || groups[0]!=12345 {panic("credential fixture groups")}
  must(os.Chmod(file,0))
  if syscall.Faccessat(fd,"file",4,0)!=syscall.EACCES {panic("real access must deny mode 0")}
  effective:=syscall.Faccessat(fd,"file",4,0x200)
  if os.Getenv("GART_FACCESSAT_DAC_OVERRIDE")=="1" {must(effective)} else if effective!=syscall.EACCES {panic("effective access must deny mode 0")}

  for _,name:=range []string{"owner","primary","supplementary","other"}{
   for _,mode:=range []uint32{0,1,2,4,7}{
    for _,flags:=range []int{0,0x100,0x200,0x300}{show(name,fd,filepath.Join(fixtures,name),mode,flags)}
   }
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

func TestLinuxFaccessatUpstream(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Faccessat requires Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/faccessatupstream\n\ngo 1.27\n")
	inventory, err := parity.Discover(build.Default)
	if err != nil {
		t.Fatal(err)
	}
	sourceDir := filepath.Join(runtime.GOROOT(), "src/syscall")
	source := selectedUpstreamSource(t, filepath.Join(sourceDir, "syscall_linux_test.go"), []string{"touch", "_AT_SYMLINK_NOFOLLOW", "TestFaccessat"})
	writeFile(t, filepath.Join(dir, "faccessat_test.go"), source)
	license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
	t.Log(run(t, dir, "go", "test", "-v", "."))
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true})
	if err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			args := upstreamTestArgs("-test.run=^TestFaccessat$")
			var got programResult
			if target == "vm" {
				got = upstreamOutcome(t, dir, nil, runLimit, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				got = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstreamInventory(t, "syscall-faccessat", "syscall", "^TestFaccessat$", target, sourceDir, got.Text, got.Exit, *inventory, nil, nil, nil, true)
		})
	}
}
