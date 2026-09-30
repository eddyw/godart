package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestExecutableAccess(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native executable access requires macOS or Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/executableaccess\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"os";"os/exec";"path/filepath";"syscall")
func main(){
 root,err:=os.MkdirTemp("","execaccess");if err!=nil{panic(err)};defer os.RemoveAll(root)
 report:=func(label,path string){
  found,err:=exec.LookPath(path);relative:="";if found!=""{relative,_=filepath.Rel(root,found)}
  var failure *exec.Error;kind:="nil";if errors.As(err,&failure){kind=fmt.Sprintf("%T",failure.Err)}
  fmt.Printf("%s path=%q error=%t permission=%t directory=%t kind=%s\n",label,relative,err!=nil,errors.Is(err,syscall.EACCES),errors.Is(err,syscall.EISDIR),kind)
 }
 for _,mode:=range []os.FileMode{0,0600,0644,0700,0100,0010,0001,0111,0755}{
  label:=fmt.Sprintf("%04o",mode);name:=filepath.Join(root,label)
  if err:=os.WriteFile(name,[]byte("#!/bin/sh\nexit 0\n"),0600);err!=nil{panic(err)}
  if err:=os.Chmod(name,mode);err!=nil{panic(err)}
  report(label,name)
 }
 report("directory",root)
 for _,target:=range []string{"0600","0700"}{
  link:=filepath.Join(root,"link"+target);if err:=os.Symlink(target,link);err!=nil{panic(err)};report("link"+target,link)
 }
 first,second:=filepath.Join(root,"first"),filepath.Join(root,"second")
 for _,d:=range []string{first,second}{if err:=os.Mkdir(d,0700);err!=nil{panic(err)}}
 if err:=os.WriteFile(filepath.Join(first,"tool"),[]byte("first"),0600);err!=nil{panic(err)}
 if err:=os.WriteFile(filepath.Join(second,"tool"),[]byte("second"),0700);err!=nil{panic(err)}
 if err:=os.Setenv("PATH",first+string(os.PathListSeparator)+second);err!=nil{panic(err)}
 report("search","tool")
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

func TestExecutableLookupMissing(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/executablemissing\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"os";"os/exec")
func main(){
 if err:=os.Setenv("PATH", "");err!=nil{panic(err)}
 path,err:=exec.LookPath("absent-program")
 fmt.Printf("path=%q missing=%t\n",path,errors.Is(err,exec.ErrNotFound))
 fmt.Println("command",errors.Is(exec.Command("absent-program").Err,exec.ErrNotFound))
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
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
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
