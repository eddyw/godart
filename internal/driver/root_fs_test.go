package driver_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestRootFSContracts(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("root filesystem methods require a native filesystem")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/rootfs\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"io/fs";"os";"path/filepath")
func main(){
 dir,err:=os.MkdirTemp("","root-fs");if err!=nil{panic(err)};defer os.RemoveAll(dir)
 for _,name:=range []string{"z","a","m"}{if err:=os.WriteFile(filepath.Join(dir,name),[]byte(name),0600);err!=nil{panic(err)}}
 if err:=os.Symlink("a",filepath.Join(dir,"link"));err!=nil{panic(err)}
 root,err:=os.OpenRoot(dir);if err!=nil{panic(err)};defer root.Close()
 fsys:=root.FS()
 _,readDir:=fsys.(fs.ReadDirFS);_,readLink:=fsys.(fs.ReadLinkFS)
 fmt.Println("interfaces",readDir,readLink)
 names,err:=fs.ReadDir(fsys,".");if err!=nil{panic(err)}
 for _,entry:=range names{fmt.Println("entry",entry.Name())}
 for _,name:=range []string{"link","missing","../escape","","a"}{
  target,err:=fs.ReadLink(fsys,name)
  var pe *fs.PathError;op:="";if errors.As(err,&pe){op=pe.Op}
  fmt.Printf("readlink %q target=%q error=%t invalid=%t missing=%t op=%s\n",name,target,err!=nil,errors.Is(err,fs.ErrInvalid),errors.Is(err,fs.ErrNotExist),op)
  info,err:=fs.Lstat(fsys,name);link:=info!=nil&&info.Mode()&fs.ModeSymlink!=0
  op="";if errors.As(err,&pe){op=pe.Op}
  fmt.Printf("lstat %q link=%t error=%t invalid=%t missing=%t op=%s\n",name,link,err!=nil,errors.Is(err,fs.ErrInvalid),errors.Is(err,fs.ErrNotExist),op)
 }
 for _,name:=range []string{"missing","../escape",""}{
  f,err:=fsys.Open(name);fmt.Printf("open %q nil=%t error=%t\n",name,f==nil,err!=nil)
  _,err=fs.ReadFile(fsys,name);var pe *fs.PathError;op:="";if errors.As(err,&pe){op=pe.Op}
  fmt.Printf("readfile %q error=%t op=%s\n",name,err!=nil,op)
  _,err=fs.ReadDir(fsys,name);op="";if errors.As(err,&pe){op=pe.Op}
  fmt.Printf("readdir %q error=%t op=%s\n",name,err!=nil,op)
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

func TestRootFSUpstream(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("root filesystem methods require a native filesystem")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/rootfs\n\ngo 1.27\n")
	names := []string{"TestRootDirFS", "testDirFS", "forceMFTUpdateOnWindows"}
	pattern := "^TestRootDirFS$"
	source := selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/os/os_test.go"), names)
	writeFile(t, filepath.Join(dir, "rootfs_test.go"), source)
	license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
	if err := os.MkdirAll(filepath.Join(dir, "testdata/dirfs/dir"), 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a", "b", "dir/x"} {
		data, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "src/os/testdata/dirfs", name))
		if err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(dir, "testdata/dirfs", name), string(data))
	}
	t.Log(run(t, dir, "go", "test", "-v", "."))
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true})
	if err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	targets := []string{"vm", "aot"}
	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			targetPattern := pattern
			args := upstreamTestArgs("-test.run=" + targetPattern)
			var got programResult
			if target == "vm" {
				got = upstreamOutcome(t, dir, nil, runLimit, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				got = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "root-fs", "os", targetPattern, target, filepath.Join(runtime.GOROOT(), "src/os"), got.Text, got.Exit, true)
		})
	}
}
