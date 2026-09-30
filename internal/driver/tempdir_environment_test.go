package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestTempDirectoryEnvironment(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("this adapter covers Unix temporary directories")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/tempdir\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"os";"path/filepath";"strings")
func main(){
 for _,value:=range []string{"","/tmp/selected","relative-temp","/tmp/a/../b/"}{
  if err:=os.Setenv("TMPDIR",value);err!=nil{panic(err)}
  fmt.Printf("value=%q directory=%q\n",value,os.TempDir())
 }
 if len(os.Args)<2{return}
 root,err:=os.MkdirTemp(".","temp-environment");if err!=nil{panic(err)};defer os.RemoveAll(root)
 if err:=os.Setenv("TMPDIR",root);err!=nil{panic(err)}
 file,err:=os.CreateTemp("","file");if err!=nil{panic(err)}
 fmt.Println("file uses environment",strings.HasPrefix(file.Name(),root+string(filepath.Separator)))
 file.Close();os.Remove(file.Name())
 dir,err:=os.MkdirTemp("","dir");if err!=nil{panic(err)}
 fmt.Println("directory uses environment",strings.HasPrefix(dir,root+string(filepath.Separator)))
 os.Remove(dir)
}
`)
	want := programOutcome(t, dir, "go", "run", ".")
	if want.Exit != 0 {
		t.Fatalf("native Go failed: %+v", want)
	}
	wantFiles := programOutcome(t, dir, "go", "run", ".", "files")
	if wantFiles.Exit != 0 {
		t.Fatalf("native Go failed: %+v", wantFiles)
	}
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	targets := []string{"vm", "aot", "js", "wasm"}
	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			var got programResult
			expected := want
			args := []string{}
			if target == "vm" || target == "aot" {
				expected = wantFiles
				args = []string{"files"}
			}
			if target == "vm" {
				got = programOutcome(t, dir, "dart", filepath.Join(out, "bin/main.dart"), "files")
			} else {
				got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, args)
			}
			if got != expected {
				t.Fatalf("Go:\n%s\nGart (exit %d):\n%s", expected.Text, got.Exit, got.Text)
			}
		})
	}
}
