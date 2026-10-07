package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRootLstatErrno(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("root stat errno requires macOS or Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/rootstaterrno\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"io/fs";"os";"path/filepath";"syscall")
func main(){
 dir,err:=os.MkdirTemp("","rootstaterrno");if err!=nil{panic(err)};defer os.RemoveAll(dir)
 if err:=os.WriteFile(filepath.Join(dir,"file"),[]byte("abc"),0600);err!=nil{panic(err)}
 if err:=os.Symlink("file",filepath.Join(dir,"link"));err!=nil{panic(err)}
 root,err:=os.OpenRoot(dir);if err!=nil{panic(err)};defer root.Close()
 missing,invalid,regular,link:=0,0,0,0
 for i:=0;i<32;i++{
  _,err:=root.Lstat("missing");if errors.Is(err,fs.ErrNotExist){missing++}
  _,err=root.Lstat("bad\x00name");if errors.Is(err,syscall.EINVAL){invalid++}
  info,err:=root.Lstat("file");if err==nil&&info.Size()==3&&info.Mode().IsRegular(){regular++}
  info,err=root.Lstat("link");if err==nil&&info.Mode()&fs.ModeSymlink!=0{link++}
 }
 fmt.Println("stat calls",missing,invalid,regular,link)
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
	for _, target := range []string{"vm", "aot"} {
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
