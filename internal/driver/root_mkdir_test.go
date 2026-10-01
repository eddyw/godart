package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestRootMkdirContracts(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("root directory creation requires a native filesystem")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/rootmkdir\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"os";"path/filepath";"syscall")
func report(name string,err error){
 var pe *os.PathError
 fields:=errors.As(err,&pe)&&pe.Op=="mkdirat"&&pe.Path==name
 fmt.Printf("%q success=%t fields=%t exists=%t missing=%t notdir=%t invalid=%t closed=%t\n",name,err==nil,fields,errors.Is(err,os.ErrExist),errors.Is(err,os.ErrNotExist),errors.Is(err,syscall.ENOTDIR),errors.Is(err,syscall.EINVAL),errors.Is(err,os.ErrClosed))
}
func main(){
 base,err:=os.MkdirTemp("","root-mkdir");if err!=nil{panic(err)};defer os.RemoveAll(base)
 dir:=filepath.Join(base,"root");if err:=os.Mkdir(dir,0700);err!=nil{panic(err)}
 if err:=os.Mkdir(filepath.Join(base,"outside"),0700);err!=nil{panic(err)}
 root,err:=os.OpenRoot(dir);if err!=nil{panic(err)};defer root.Close()
 if err:=root.WriteFile("file",[]byte("safe"),0600);err!=nil{panic(err)}
 if err:=root.Mkdir("directory",0700);err!=nil{panic(err)}
 for _,pair:=range [][2]string{{"directory","link"},{"missing","dangling"},{"../outside","outward"},{"file","filelink"}}{if err:=root.Symlink(pair[0],pair[1]);err!=nil{panic(err)}}
 for _,name:=range []string{"fresh","fresh-slash/","directory","file","link","link/","dangling","dangling/","outward","outward/","filelink","filelink/","file/child","missing/child","../escape","/","","bad\x00name"}{report(name,root.Mkdir(name,0700))}
 report("mode",root.Mkdir("mode",os.ModeSticky|0700))
 root.Close()
 for _,name:=range []string{"closed","../escape","","bad\x00name"}{report(name,root.Mkdir(name,0700))}
 report("mode",root.Mkdir("mode",os.ModeSticky|0700))
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
