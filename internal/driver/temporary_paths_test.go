package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestTemporaryPathSeparators(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("temporary file creation requires a native filesystem")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/temporarypaths\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"os";"strings")
func main(){
 root,err:=os.MkdirTemp(".","temporary-paths");if err!=nil{panic(err)};defer os.RemoveAll(root)
 for _,ending:=range []string{"","/","//"}{
  dir:=root+ending
  prefix:=dir;if ending==""{prefix+="/"}
  f,err:=os.CreateTemp(dir,"file*tail");if err!=nil{panic(err)}
  fmt.Printf("file ending=%q prefix=%t suffix=%t\n",ending,strings.HasPrefix(f.Name(),prefix+"file"),strings.HasSuffix(f.Name(),"tail"))
  f.Close();os.Remove(f.Name())
  name,err:=os.MkdirTemp(dir,"dir*tail");if err!=nil{panic(err)}
  fmt.Printf("directory ending=%q prefix=%t suffix=%t\n",ending,strings.HasPrefix(name,prefix+"dir"),strings.HasSuffix(name,"tail"))
  os.Remove(name)
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
