package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestRootExclusiveCreate(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("root-open errors require a native filesystem")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/rootexclusive\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"os";"path/filepath";"syscall")
func report(root *os.Root,name string){
 f,err:=root.OpenFile(name,os.O_CREATE|os.O_EXCL|os.O_WRONLY,0600)
 named:=f!=nil&&f.Name()==root.Name()+"/"+name
 if f!=nil{f.Close()}
 var pe *os.PathError
 fmt.Printf("%q file=%t named=%t fields=%t exists=%t loop=%t closed=%t\n",name,f!=nil,named,errors.As(err,&pe)&&pe.Op=="openat"&&pe.Path==name,errors.Is(err,syscall.EEXIST),errors.Is(err,syscall.ELOOP),errors.Is(err,os.ErrClosed))
}
func main(){
 base,err:=os.MkdirTemp("","root-exclusive");if err!=nil{panic(err)};defer os.RemoveAll(base)
 dir:=filepath.Join(base,"root");if err:=os.Mkdir(dir,0700);err!=nil{panic(err)}
 outside:=filepath.Join(base,"outside");if err:=os.WriteFile(outside,[]byte("safe"),0600);err!=nil{panic(err)}
 root,err:=os.OpenRoot(dir);if err!=nil{panic(err)};defer root.Close()
 if err:=root.WriteFile("file",[]byte("safe"),0600);err!=nil{panic(err)}
 for _,pair:=range [][2]string{{"file","link"},{"missing","dangling"},{"../outside","outward"},{"../absent","outside-missing"},{"loop","loop"}}{if err:=root.Symlink(pair[0],pair[1]);err!=nil{panic(err)}}
 for _,name:=range []string{"fresh","file","link","dangling","outward","outside-missing","loop","loop/","outward/","missing/child","../escape"}{report(root,name)}
 data,err:=os.ReadFile(outside);fmt.Printf("outside=%q error=%v\n",data,err)
 data,err=root.ReadFile("file");fmt.Printf("file=%q error=%v\n",data,err)
 root.Close();report(root,"closed")
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
