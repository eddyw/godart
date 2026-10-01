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

func TestLinuxFchmodat(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Fchmodat requires Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/fchmodat\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"os";"path/filepath";"syscall")
func must(err error){if err!=nil{panic(err)}}
func main(){
 parent,err:=os.MkdirTemp("","fchmodat-");must(err);defer os.RemoveAll(parent)
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
  err:=syscall.Fchmodat(base,path,mode,flags)
  code:=syscall.Errno(0)
  if err!=nil{var ok bool;code,ok=err.(syscall.Errno);if !ok{panic(err)}}
  info,err:=os.Lstat(file);must(err)
  fmt.Println(label,uint64(code),uint32(info.Mode()))
 }
 for _,mode:=range []uint32{0,0640,0755,04755,02755,01777,0xffffffff}{show("mode",fd,"file",mode,0)}
 for _,item:=range []struct{label,path string;flags int}{
  {"dot","./file",0},{"parent","sub/../file",0},{"follow","link",0},
  {"nofollow file","file",0x100},{"nofollow link","link",0x100},
  {"dangling","dangling",0},{"dangling nofollow","dangling",0x100},
  {"absent","absent",0},{"not-directory","file/child",0},
  {"raw",raw,0},{"empty","",0},{"nul","file\x00tail",0},
  {"invalid flags","file",1},{"invalid effective","file",0x200},
  {"invalid all flags","file",-1},{"invalid absent","absent",1},
  {"invalid nul","file\x00tail",1},
 }{show(item.label,fd,item.path,0644,item.flags)}
 show("empty file descriptor",filefd,"",0600,0x1000)
 show("empty file nofollow",filefd,"",0640,0x1100)
 show("invalid descriptor",-1,"file",0644,0)
 show("absolute ignores descriptor",-1,file,0600,0)
 show("absolute nofollow",-1,file,0640,0x100)
 show("invalid descriptor flags",-1,"file",0644,1)
 show("invalid descriptor nul",-1,"file\x00tail",0644,0)
 closed,err:=syscall.Dup(fd);must(err);must(syscall.Close(closed))
 show("closed descriptor",closed,"file",0644,0)
 info,err:=os.Stat(filepath.Join(moved,raw));must(err)
 fmt.Println("raw mode",info.Mode().Perm())
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

func TestLinuxFchmodatUpstream(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Fchmodat requires Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/fchmodatupstream\n\ngo 1.27\n")
	inventory, err := parity.Discover(build.Default)
	if err != nil {
		t.Fatal(err)
	}
	sourceDir := filepath.Join(runtime.GOROOT(), "src/syscall")
	source := selectedUpstreamSource(t, filepath.Join(sourceDir, "syscall_linux_test.go"), []string{"touch", "_AT_SYMLINK_NOFOLLOW", "TestFchmodat"})
	writeFile(t, filepath.Join(dir, "fchmodat_test.go"), source)
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
			args := upstreamTestArgs("-test.run=^TestFchmodat$")
			var got programResult
			if target == "vm" {
				got = upstreamOutcome(t, dir, nil, runLimit, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				got = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstreamInventory(t, "syscall-fchmodat", "syscall", "^TestFchmodat$", target, sourceDir, got.Text, got.Exit, *inventory, nil, nil, nil, true)
		})
	}
}
