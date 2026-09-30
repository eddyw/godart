package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestRootHardLinks(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("root hard links require a native filesystem")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/roothardlinks\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"os";"path/filepath")
func report(old,new string,err error){
 var le *os.LinkError
 fields:=errors.As(err,&le)&&le.Op=="linkat"&&le.Old==old&&le.New==new
 fmt.Printf("link %q %q success=%t fields=%t exists=%t missing=%t closed=%t\n",old,new,err==nil,fields,errors.Is(err,os.ErrExist),errors.Is(err,os.ErrNotExist),errors.Is(err,os.ErrClosed))
}
func main(){
 dir,err:=os.MkdirTemp("","root-hardlinks");if err!=nil{panic(err)};defer os.RemoveAll(dir)
 if err:=os.WriteFile(filepath.Join(dir,"file"),[]byte("data"),0600);err!=nil{panic(err)}
 for _,entry:=range [][2]string{{"sym","file"},{"dangling","absent"},{"outward","../outside"}}{
  if err:=os.Symlink(entry[1],filepath.Join(dir,entry[0]));err!=nil{panic(err)}
 }
 root,err:=os.OpenRoot(dir);if err!=nil{panic(err)};defer root.Close()
 for _,name:=range []string{"file","sym","dangling","outward"}{
  err:=root.Link(name,"copy");report(name,"copy",err)
  if err==nil{
   a,e:=os.Lstat(filepath.Join(dir,name));if e!=nil{panic(e)}
   b,e:=os.Lstat(filepath.Join(dir,"copy"));if e!=nil{panic(e)}
   target,_:=root.Readlink("copy")
   fmt.Printf("same=%t type=%t target=%q\n",os.SameFile(a,b),a.Mode().Type()==b.Mode().Type(),target)
   if e:=root.Remove("copy");e!=nil{panic(e)}
  }
 }
 for _,pair:=range [][2]string{{"missing","copy"},{"file","file"},{"../outside","copy"},{"file","../outside"},{"file","bad\x00name"}}{report(pair[0],pair[1],root.Link(pair[0],pair[1]))}
 root.Close();report("file","copy",root.Link("file","copy"))
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
