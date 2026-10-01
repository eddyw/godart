package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestRemoveAllErrors(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("removal requires a native filesystem")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/removeallerrors\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"os";"path/filepath";"strings";"syscall")
func main(){
 if len(os.Args)>1 {
  name:="unavailable-removeall";err:=os.RemoveAll(name);var pe *os.PathError
  fmt.Printf("unavailable=%t fields=%t\n",err!=nil,errors.As(err,&pe)&&pe.Op=="removeall"&&pe.Path==name)
  return
 }
 base,err:=os.MkdirTemp("","removeall-errors");if err!=nil{panic(err)};defer os.RemoveAll(base)
 for _,rooted:=range []bool{false,true}{
  dir:=filepath.Join(base,fmt.Sprint(rooted));if err:=os.Mkdir(dir,0700);err!=nil{panic(err)}
  root,err:=os.OpenRoot(dir);if err!=nil{panic(err)}
  if err:=root.WriteFile("file",[]byte("safe"),0600);err!=nil{panic(err)}
  if err:=root.Mkdir("locked",0700);err!=nil{panic(err)}
  if err:=root.WriteFile("locked/child",[]byte("safe"),0600);err!=nil{panic(err)}
  if err:=root.Chmod("locked",0);err!=nil{panic(err)}
  for _,name:=range []string{"file/child","locked/child","locked","bad\x00name","bad\x00parent/child","missing/bad\x00name","missing/child"}{
   var err error
   if rooted{err=root.RemoveAll(name)}else{err=os.RemoveAll(dir+"/"+name)}
   var pe *os.PathError;op,path:="",""
   if errors.As(err,&pe){op=pe.Op;path=strings.TrimPrefix(pe.Path,dir+"/")}
   fmt.Printf("root=%t %q success=%t op=%q path=%q notdir=%t access=%t invalid=%t\n",rooted,name,err==nil,op,path,errors.Is(err,syscall.ENOTDIR),errors.Is(err,syscall.EACCES),errors.Is(err,syscall.EINVAL))
  }
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
	webWant := programResult{Exit: 0, Text: "unavailable=true fields=true"}
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
