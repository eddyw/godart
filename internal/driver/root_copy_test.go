package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestRootCopyState(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("root state validation requires a native filesystem")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/rootcopy\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"io/fs";"os";"path/filepath")
func main(){
 dir,err:=os.MkdirTemp("","root-copy");if err!=nil{panic(err)};defer os.RemoveAll(dir)
 if err:=os.WriteFile(filepath.Join(dir,"file"),[]byte("original"),0600);err!=nil{panic(err)}
 for _,closeCopy:=range []bool{false,true}{
  original,err:=os.OpenRoot(dir);if err!=nil{panic(err)}
  copied:=*original;view:=copied.FS()
  independent,err:=os.OpenRoot(dir);if err!=nil{panic(err)}
  if closeCopy{err=copied.Close()}else{err=original.Close()};if err!=nil{panic(err)}
  for _,r:=range []*os.Root{original,&copied}{
   f,err:=r.Open("file");if f!=nil{f.Close()}
   fmt.Println("closed",closeCopy,errors.Is(err,os.ErrClosed),r.Name()==dir)
  }
  _,err=fs.Stat(view,"file");fmt.Println("view closed",errors.Is(err,os.ErrClosed))
  data,err:=independent.ReadFile("file");fmt.Printf("independent=%q error=%v\n",data,err)
  original.Close();copied.Close();independent.Close()
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
