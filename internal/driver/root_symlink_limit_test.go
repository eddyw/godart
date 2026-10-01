package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestRootSymlinkLimit(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("root-open errors require a native filesystem")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/rootsymlinklimit\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"os";"syscall")
func report(op string,err error){fmt.Printf("%s success=%t loop=%t\n",op,err==nil,errors.Is(err,syscall.ELOOP))}
func main(){
 dir,err:=os.MkdirTemp("","root-symlinks");if err!=nil{panic(err)};defer os.RemoveAll(dir)
 root,err:=os.OpenRoot(dir);if err!=nil{panic(err)};defer root.Close()
 if err:=root.Mkdir("target",0700);err!=nil{panic(err)}
 if err:=root.WriteFile("target/file",[]byte("safe"),0600);err!=nil{panic(err)}
 target:="target"
 for i:=1;i<=9;i++{name:=fmt.Sprintf("link%d",i);if err:=root.Symlink(target,name);err!=nil{panic(err)};target=name}
 if err:=root.Symlink("loop","loop");err!=nil{panic(err)}
 for _,name:=range []string{"link8","link9","loop"}{
  fmt.Println(name)
  _,err:=root.Stat(name);report("stat",err)
  _,err=root.Lstat(name);report("lstat",err)
  child,err:=root.OpenRoot(name);if child!=nil{child.Close()};report("openroot",err)
  _,err=root.ReadFile(name+"/file");report("readfile",err)
  report("mkdirall",root.MkdirAll(name,0700))
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
