package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestRootMkdirAllContracts(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("root directory creation requires a native filesystem")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/rootmkdirall\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"os";"path/filepath";"syscall")
func main(){
 base,err:=os.MkdirTemp("","root-mkdirall");if err!=nil{panic(err)};defer os.RemoveAll(base)
 for index,name:=range []string{"fresh/child","locked","locked/child","directory","file","file/child","link","link/child","dangling","dangling/","dangling/child","outward","outward/","outward/child","filelink","filelink/","filelink/child","new/../other","new/../../escape","../escape","/","","bad\x00name","bad\x00parent/child"}{
  dir:=filepath.Join(base,fmt.Sprint(index));if err:=os.Mkdir(dir,0700);err!=nil{panic(err)}
  root,err:=os.OpenRoot(dir);if err!=nil{panic(err)}
  if err:=root.WriteFile("file",[]byte("safe"),0600);err!=nil{panic(err)}
  if err:=root.Mkdir("directory",0700);err!=nil{panic(err)}
  for _,pair:=range [][2]string{{"directory","link"},{"missing","dangling"},{"..","outward"},{"file","filelink"}}{if err:=root.Symlink(pair[0],pair[1]);err!=nil{panic(err)}}
  if err:=root.Mkdir("locked",0700);err!=nil{panic(err)}
  if err:=root.Chmod("locked",0);err!=nil{panic(err)}
  err=root.MkdirAll(name,0700)
  var pe *os.PathError;op,path:="",""
  if errors.As(err,&pe){op=pe.Op;path=pe.Path}
  _,missingErr:=root.Stat("missing");_,newErr:=root.Stat("new")
  fmt.Printf("%q success=%t op=%q path=%q exists=%t notdir=%t invalid=%t access=%t made-target=%t made-parent=%t\n",name,err==nil,op,path,errors.Is(err,syscall.EEXIST),errors.Is(err,syscall.ENOTDIR),errors.Is(err,syscall.EINVAL),errors.Is(err,syscall.EACCES),missingErr==nil,newErr==nil)
  os.Chmod(dir+"/locked",0700)
  root.Close()
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
