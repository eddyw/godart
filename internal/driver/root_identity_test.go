package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestRootDescriptorIdentity(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("root-open errors require a native filesystem")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/rootidentity\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"io";"io/fs";"os";"path/filepath";"time";"runtime")
func must(err error){if err!=nil{panic(err)}}
func main(){
 base,err:=os.MkdirTemp("","root-identity");must(err);defer os.RemoveAll(base)
 old,moved:=filepath.Join(base,"old"),filepath.Join(base,"moved")
 must(os.MkdirAll(old+"/sub",0700));must(os.WriteFile(old+"/marker",[]byte("original"),0600));must(os.WriteFile(old+"/sub/marker",[]byte("child"),0600))
 raw:="raw-name";if runtime.GOOS=="linux"{raw="raw-\xff"}
 root,err:=os.OpenRoot(old);must(err);defer root.Close();copy:=*root
 child,err:=root.OpenRoot("sub");must(err);defer child.Close()
 must(os.Rename(old,moved));must(os.Mkdir(old,0700));must(os.WriteFile(old+"/marker",[]byte("decoy"),0600))
 data,err:=root.ReadFile("marker");must(err);fmt.Printf("read=%q name=%t\n",data,root.Name()==old)
 must(root.WriteFile("marker",[]byte("changed"),0600))
 must(root.Mkdir("new",0700));must(root.MkdirAll("tree/a/b",0700))
 must(root.WriteFile(raw,[]byte("raw"),0600));must(root.Symlink(raw,"link"))
 target,err:=root.Readlink("link");must(err);fmt.Printf("link=%q\n",target)
 must(root.Link("link","hardlink"));first,err:=root.Lstat("link");must(err);second,err:=root.Lstat("hardlink");must(err);fmt.Println("hardlink",os.SameFile(first,second))
 must(root.Chmod("link",0400));info,err:=root.Stat(raw);must(err);fmt.Printf("mode=%o\n",info.Mode().Perm())
 must(root.Chown("link",-1,-1));must(root.Lchown("link",-1,-1))
 stamp:=time.Unix(1234567890,123456789);must(root.Chtimes("link",stamp,stamp));info,err=root.Stat(raw);must(err);fmt.Println("time",info.ModTime().Equal(stamp))
 must(root.Rename(raw,"renamed"));must(root.Remove("hardlink"));must(root.RemoveAll("tree"))
 entries,err:=root.FS().(fs.ReadDirFS).ReadDir(".");must(err);for _,entry:=range entries{fmt.Println("entry",entry.Name())}
 must(os.Rename(moved+"/sub",moved+"/sub-moved"));must(os.Mkdir(moved+"/sub",0700));must(os.WriteFile(moved+"/sub/marker",[]byte("child decoy"),0600))
 data,err=child.ReadFile("marker");must(err);fmt.Printf("child=%q name=%t\n",data,child.Name()==old+"/sub")
 held,err:=root.Open("marker");must(err);defer held.Close();must(root.Close())
 _,err=copy.Stat("marker");fmt.Println("copy closed",errors.Is(err,os.ErrClosed))
 data,err=io.ReadAll(held);must(err);fmt.Printf("held=%q name=%t\n",data,held.Name()==old+"/marker")
 data,err=os.ReadFile(old+"/marker");must(err);fmt.Printf("decoy=%q\n",data)
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
