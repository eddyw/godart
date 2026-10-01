package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestRemovalErrors(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("removal requires a native filesystem")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/removalerrors\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"os";"path/filepath";"syscall")
func report(op,name string,err error){
 var pe *os.PathError
 fields:=errors.As(err,&pe)&&pe.Op==op&&pe.Path==name
 fmt.Printf("%s %q success=%t fields=%t missing=%t notdir=%t nonempty=%t invalid=%t closed=%t\n",op,name,err==nil,fields,errors.Is(err,syscall.ENOENT),errors.Is(err,syscall.ENOTDIR),errors.Is(err,syscall.ENOTEMPTY),errors.Is(err,syscall.EINVAL),errors.Is(err,os.ErrClosed))
}
func main(){
 if len(os.Args)>1 {
  err:=os.Remove("");var pe *os.PathError
  fmt.Printf("empty failure=%t fields=%t\n",err!=nil,errors.As(err,&pe)&&pe.Op=="remove"&&pe.Path=="")
  return
 }
 base,err:=os.MkdirTemp("","removal-errors");if err!=nil{panic(err)};defer os.RemoveAll(base)
 for _,rooted:=range []bool{false,true}{
  dir:=filepath.Join(base,fmt.Sprint(rooted));if err:=os.Mkdir(dir,0700);err!=nil{panic(err)}
  root,err:=os.OpenRoot(dir);if err!=nil{panic(err)}
  if err:=root.WriteFile("file",[]byte("safe"),0600);err!=nil{panic(err)}
  if err:=root.MkdirAll("nonempty/child",0700);err!=nil{panic(err)}
  if err:=root.Mkdir("empty",0700);err!=nil{panic(err)}
  if err:=root.Symlink("file","link");err!=nil{panic(err)}
  if err:=root.Symlink("missing","dangling");err!=nil{panic(err)}
  for _,name:=range []string{"missing","nonempty","file/child","bad\x00name","link","dangling","empty","file"}{
   if rooted {report("removeat",name,root.Remove(name))}else{full:=dir+"/"+name;report("remove",full[len(dir)+1:],normalize(os.Remove(full),full,name))}
  }
  if rooted {
   for _,name:=range []string{"../escape",""}{report("removeat",name,root.Remove(name));report("RemoveAll",name,root.RemoveAll(name))}
   root.Close()
   for _,name:=range []string{"closed","../escape","","bad\x00name"}{report("removeat",name,root.Remove(name));report("RemoveAll",name,root.RemoveAll(name))}
  }else{root.Close()}
 }
}
func normalize(err error,full,name string)error{if pe,ok:=err.(*os.PathError);ok&&pe.Path==full{pe.Path=name};return err}

`)
	want := programOutcome(t, dir, "go", "run", ".")
	if want.Exit != 0 {
		t.Fatalf("native Go failed: %+v", want)
	}
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	webWant := programOutcome(t, dir, "go", "run", ".", "web")
	if webWant.Exit != 0 {
		t.Fatalf("native empty-path call failed: %+v", webWant)
	}
	targets := []string{"vm", "aot", "js", "wasm"}
	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			var got programResult
			expected := want
			args := []string{}
			if target == "js" || target == "wasm" {
				expected = webWant
				args = []string{"web"}
			}
			if target == "vm" {
				got = programOutcome(t, dir, "dart", filepath.Join(out, "bin/main.dart"))
			} else {
				got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, args)
			}
			if got != expected {
				t.Fatalf("Go:\n%s\nGart (exit %d):\n%s", expected.Text, got.Exit, got.Text)
			}
		})
	}
}
