package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestChildRootErrors(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("root-open errors require a native filesystem")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/childrooterrors\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"os";"path/filepath";"syscall")
func report(root *os.Root,name string){
 child,err:=root.OpenRoot(name)
 op,path:="",""
 var pe *os.PathError
 if errors.As(err,&pe){op=pe.Op;if pe.Path==name{path="relative"}else if pe.Path==root.Name()+"/"+name{path="joined"}else{path="other"}}
 named:=child!=nil&&child.Name()==root.Name()+"/"+name
 if child!=nil{child.Close()}
 fmt.Printf("%q root=%t named=%t op=%q path=%q missing=%t invalid=%t notdir=%t closed=%t\n",name,child!=nil,named,op,path,os.IsNotExist(err),errors.Is(err,syscall.EINVAL),errors.Is(err,syscall.ENOTDIR),errors.Is(err,os.ErrClosed))
}
func main(){
 base,err:=os.MkdirTemp("","child-root");if err!=nil{panic(err)};defer os.RemoveAll(base)
 dir:=filepath.Join(base,"root");if err:=os.Mkdir(dir,0700);err!=nil{panic(err)}
 root,err:=os.OpenRoot(dir);if err!=nil{panic(err)};defer root.Close()
 if err:=root.WriteFile("file",[]byte("safe"),0600);err!=nil{panic(err)}
 if err:=root.Mkdir("directory",0700);err!=nil{panic(err)}
 for _,pair:=range [][2]string{{"directory","link"},{"missing","dangling"},{"..","outward"},{"file","filelink"}}{if err:=root.Symlink(pair[0],pair[1]);err!=nil{panic(err)}}
 for _,name:=range []string{"directory","directory/",".","link","file","file/","file/child","filelink","filelink/","missing","missing/child","dangling","outward","../escape","/","","bad\x00name"}{report(root,name)}
 root.Close()
 for _,name:=range []string{"directory","../escape","","bad\x00name"}{report(root,name)}
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
