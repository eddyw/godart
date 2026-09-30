package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestRootPathMutations(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("root path mutation requires a native filesystem")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/rootmutations\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"os";"path/filepath")
func report(op,old,new string,err error){
 var le *os.LinkError
 fields:=errors.As(err,&le)&&le.Op==op+"at"&&le.Old==old&&le.New==new
 fmt.Printf("%s %q %q success=%t fields=%t exists=%t missing=%t closed=%t\n",op,old,new,err==nil,fields,errors.Is(err,os.ErrExist),errors.Is(err,os.ErrNotExist),errors.Is(err,os.ErrClosed))
}
func main(){
 base,err:=os.MkdirTemp("","root-mutations");if err!=nil{panic(err)};defer os.RemoveAll(base)
 dir:=filepath.Join(base,"root");if err:=os.Mkdir(dir,0700);err!=nil{panic(err)}
 outside:=filepath.Join(base,"outside");if err:=os.WriteFile(outside,[]byte("safe"),0600);err!=nil{panic(err)}
 root,err:=os.OpenRoot(dir);if err!=nil{panic(err)};defer root.Close()
 if err:=root.WriteFile("source",[]byte("data"),0600);err!=nil{panic(err)}
 for _,name:=range []string{"outward","destination","occupied"}{if err:=root.Symlink("../outside",name);err!=nil{panic(err)}}
 report("rename","outward","moved",root.Rename("outward","moved"))
 target,_:=os.Readlink(filepath.Join(dir,"moved"));fmt.Printf("moved target=%q\n",target)
 report("rename","source","destination",root.Rename("source","destination"))
 data,err:=os.ReadFile(filepath.Join(dir,"destination"));fmt.Printf("destination=%q error=%v\n",data,err)
 report("symlink","anything","occupied",root.Symlink("anything","occupied"))
 for _,pair:=range [][2]string{{"missing","copy"},{"destination","missing/child"},{"../outside","copy"},{"destination","../outside"}}{report("rename",pair[0],pair[1],root.Rename(pair[0],pair[1]))}
 for _,name:=range []string{"missing/child","../escape"}{report("symlink","target",name,root.Symlink("target",name))}
 data,err=os.ReadFile(outside);fmt.Printf("outside=%q error=%v\n",data,err)
 root.Close()
 report("rename","destination","copy",root.Rename("destination","copy"))
 report("symlink","target","copy",root.Symlink("target","copy"))
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
