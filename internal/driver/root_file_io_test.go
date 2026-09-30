package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestRootFileIO(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("root file I/O requires a native filesystem")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/rootfileio\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"os";"path/filepath")
func report(label string,err error){
 var pe *os.PathError
 op,path,cause:="","",""
 if errors.As(err,&pe){op,path,cause=pe.Op,pe.Path,pe.Err.Error()}
 fmt.Printf("%s op=%q path=%q cause=%q missing=%t closed=%t\n",label,op,path,cause,os.IsNotExist(err),errors.Is(err,os.ErrClosed))
}
func main(){
 dir,err:=os.MkdirTemp("","root-fileio");if err!=nil{panic(err)};defer os.RemoveAll(dir)
 root,err:=os.OpenRoot(dir);if err!=nil{panic(err)};defer root.Close()
 if err:=root.WriteFile("file",[]byte("original"),0600);err!=nil{panic(err)}
 data,err:=root.ReadFile("file");fmt.Printf("read=%q error=%v\n",data,err)
 for _,name:=range []string{"missing/child","../escape",""}{
  _,err:=root.ReadFile(name);report("read "+name,err)
  report("write "+name,root.WriteFile(name,[]byte("data"),0600))
 }
 report("mode",root.WriteFile("file",[]byte("changed"),os.ModeSticky|0600))
 data,err=os.ReadFile(filepath.Join(dir,"file"));fmt.Printf("after mode=%q error=%v\n",data,err)
 root.Close()
 _,err=root.ReadFile("file");report("closed read",err)
 report("closed write",root.WriteFile("file",nil,0600))
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
