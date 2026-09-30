package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestRootInvalidPaths(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("root path validation requires a native filesystem")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/rootinvalid\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"os";"syscall";"time")
func main(){
 dir,err:=os.MkdirTemp("","root-invalid");if err!=nil{panic(err)};defer os.RemoveAll(dir)
 calls:=[]struct{name string;call func(*os.Root,string)error}{
  {"Open",func(r *os.Root,n string)error{f,e:=r.Open(n);if f!=nil{f.Close()};return e}},
  {"OpenFile",func(r *os.Root,n string)error{f,e:=r.OpenFile(n,os.O_RDWR,0600);if f!=nil{f.Close()};return e}},
  {"Create",func(r *os.Root,n string)error{f,e:=r.Create(n);if f!=nil{f.Close()};return e}},
  {"OpenRoot",func(r *os.Root,n string)error{f,e:=r.OpenRoot(n);if f!=nil{f.Close()};return e}},
  {"ReadFile",func(r *os.Root,n string)error{_,e:=r.ReadFile(n);return e}},
  {"WriteFile",func(r *os.Root,n string)error{return r.WriteFile(n,nil,0600)}},
  {"Stat",func(r *os.Root,n string)error{_,e:=r.Stat(n);return e}},
  {"Lstat",func(r *os.Root,n string)error{_,e:=r.Lstat(n);return e}},
  {"Readlink",func(r *os.Root,n string)error{_,e:=r.Readlink(n);return e}},
  {"Mkdir",func(r *os.Root,n string)error{return r.Mkdir(n,0700)}},
  {"MkdirAll",func(r *os.Root,n string)error{return r.MkdirAll(n,0700)}},
  {"Remove",func(r *os.Root,n string)error{return r.Remove(n)}},
  {"RemoveAll",func(r *os.Root,n string)error{return r.RemoveAll(n)}},
  {"RenameOld",func(r *os.Root,n string)error{return r.Rename(n,"copy")}},
  {"RenameNew",func(r *os.Root,n string)error{return r.Rename("file",n)}},
  {"LinkOld",func(r *os.Root,n string)error{return r.Link(n,"copy")}},
  {"LinkNew",func(r *os.Root,n string)error{return r.Link("file",n)}},
  {"Symlink",func(r *os.Root,n string)error{return r.Symlink("file",n)}},
  {"Chmod",func(r *os.Root,n string)error{return r.Chmod(n,0600)}},
  {"Chown",func(r *os.Root,n string)error{return r.Chown(n,-1,-1)}},
  {"Lchown",func(r *os.Root,n string)error{return r.Lchown(n,-1,-1)}},
  {"Chtimes",func(r *os.Root,n string)error{return r.Chtimes(n,time.Time{},time.Time{})}},
 }
 for _,closed:=range []bool{false,true}{
  root,err:=os.OpenRoot(dir);if err!=nil{panic(err)}
  if err:=root.WriteFile("file",[]byte("data"),0600);err!=nil{panic(err)}
  if closed{root.Close()}
  for _,name:=range []string{"bad\x00name","bad\x00parent/file"}{
   for _,c:=range calls{err:=c.call(root,name);fmt.Printf("%s closed=%t path=%q invalid=%t closure=%t\n",c.name,closed,name,errors.Is(err,syscall.EINVAL),errors.Is(err,os.ErrClosed))}
  }
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
