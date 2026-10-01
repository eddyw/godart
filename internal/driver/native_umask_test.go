package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestNativeUmask(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native umask requires a native filesystem")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/nativeumask\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"os";"path/filepath";"syscall")
func must(err error) { if err != nil { panic(err) } }
func main() {
 dir,err:=os.MkdirTemp("","native-umask-");must(err);defer os.RemoveAll(dir)
 original:=syscall.Umask(0022);defer syscall.Umask(original)
 previous:=0022
 for i,mask:=range []int{0077,0027,0000,0777,07777,-1,0022} {
  old:=syscall.Umask(mask)
  fmt.Println("previous",old==previous)
  previous=mask&0777
  path:=filepath.Join(dir,fmt.Sprint(i))
  file,err:=os.OpenFile(path,os.O_CREATE|os.O_EXCL|os.O_RDWR,0666);must(err)
  info,err:=file.Stat();must(err);must(file.Close())
  fmt.Printf("file %o\n",info.Mode().Perm())
  must(os.Mkdir(path+"-dir",0777))
  info,err=os.Stat(path+"-dir");must(err)
  fmt.Printf("directory %o\n",info.Mode().Perm())
 }
 fmt.Println("restore previous",syscall.Umask(original)==previous)
 fmt.Println("restored",syscall.Umask(original)==original)
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
