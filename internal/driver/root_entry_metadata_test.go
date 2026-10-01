package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestRootEntryMetadata(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("root-open errors require a native filesystem")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/rootentrymetadata\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"io/fs";"os";"slices")
func must(err error){if err!=nil{panic(err)}}
func report(label string,entries []os.DirEntry,original os.FileInfo){
 slices.SortFunc(entries,func(a,b os.DirEntry)int{if a.Name()<b.Name(){return -1};if a.Name()>b.Name(){return 1};return 0})
 for _,entry:=range entries{
  info,err:=entry.Info();must(err)
  fmt.Printf("%s %s size=%d link=%t original=%t\n",label,entry.Name(),info.Size(),info.Mode()&os.ModeSymlink!=0,os.SameFile(info,original))
 }
}
func main(){
 base,err:=os.MkdirTemp("","root-entry");must(err);defer os.RemoveAll(base)
 old,moved:=base+"/old",base+"/moved"
 must(os.MkdirAll(old+"/sub",0700));must(os.WriteFile(old+"/sub/file",[]byte("original"),0600));must(os.Symlink("file",old+"/sub/link"))
 original,err:=os.Stat(old+"/sub/file");must(err)
 root,err:=os.OpenRoot(old);must(err);defer root.Close()
 cached,err:=root.FS().(fs.ReadDirFS).ReadDir("sub");must(err)
 f,err:=root.Open("sub");must(err);defer f.Close()
 must(os.Rename(old,moved));must(os.MkdirAll(old+"/sub",0700));must(os.WriteFile(old+"/sub/file",[]byte("replacement data"),0600));must(os.WriteFile(old+"/sub/link",[]byte("replacement link"),0600))
 entries,err:=f.ReadDir(-1);must(err);must(f.Close());must(root.Close())
 must(os.WriteFile(moved+"/sub/file",[]byte("changed after read"),0600))
 report("cached",cached,original);report("renamed",entries,original)
 ordinary,err:=os.Open(moved+"/sub");must(err)
 lazy,err:=ordinary.ReadDir(-1);must(err);must(ordinary.Close())
 must(os.WriteFile(moved+"/sub/file",[]byte("lazy"),0600))
 report("ordinary",lazy,original)
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
