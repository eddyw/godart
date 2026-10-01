package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestRootOpenPermissions(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("root-open errors require a native filesystem")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/rootpermissions\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"os";"path/filepath";"syscall")
func main(){
 base,err:=os.MkdirTemp("","root-permissions");if err!=nil{panic(err)};defer os.RemoveAll(base)
 parent,err:=os.OpenRoot(base);if err!=nil{panic(err)};defer parent.Close()
 for _,mode:=range []os.FileMode{0,0100,0400,0500,0700}{
  name:=fmt.Sprintf("mode-%o",mode);full:=filepath.Join(base,name)
  if err:=os.Mkdir(full,0700);err!=nil{panic(err)}
  if err:=os.Chmod(full,mode);err!=nil{panic(err)}
  for _,rooted:=range []bool{false,true}{
   var root *os.Root;var err error;op,path:="open",full
   if rooted{root,err=parent.OpenRoot(name);op,path="openat",name}else{root,err=os.OpenRoot(full)}
   var pe *os.PathError
   fields:=errors.As(err,&pe)&&pe.Op==op&&pe.Path==path
   fmt.Printf("mode=%o root=%t success=%t fields=%t access=%t\n",mode,rooted,root!=nil,fields,errors.Is(err,syscall.EACCES))
   if root!=nil{root.Close()}
  }
  if err:=os.Chmod(full,0700);err!=nil{panic(err)}
 }
 descriptor:=func()uintptr{f,err:=os.Open(base);if err!=nil{panic(err)};fd:=f.Fd();f.Close();return fd}
 before:=descriptor()
 for i:=0;i<32;i++{
  root,err:=os.OpenRoot(base);if err!=nil{panic(err)};root.Close()
  child,err:=parent.OpenRoot("mode-700");if err!=nil{panic(err)};child.Close()
 }
 fmt.Printf("descriptors-stable=%t\n",before==descriptor())
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
