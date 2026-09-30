package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestRootOpenErrors(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("root-open errors require a native filesystem")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/rooterrors\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"os";"path/filepath";"syscall")
func main(){
 root,err:=os.MkdirTemp("","root-errors");if err!=nil{panic(err)};defer os.RemoveAll(root)
 if err:=os.WriteFile(filepath.Join(root,"file"),nil,0600);err!=nil{panic(err)}
 for _,name:=range []string{"missing","missing/child","file","file/child","bad\x00path"}{
  path:=filepath.Join(root,name)
  r,err:=os.OpenRoot(path)
  if r!=nil{r.Close()}
  var pe *os.PathError
  fields:=errors.As(err,&pe)&&pe.Op=="open"&&pe.Path==path
  nested:=false
  if pe!=nil{var inner *os.PathError;nested=errors.As(pe.Err,&inner)}
  fmt.Printf("name=%q root=%t patherror=%t nested=%t missing=%t invalid=%t notdir=%t\n",name,r!=nil,fields,nested,os.IsNotExist(err),errors.Is(err,syscall.EINVAL),errors.Is(err,syscall.ENOTDIR))
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
