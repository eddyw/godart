package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCommandStartErrors(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native process errors require macOS or Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/starterrors\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"os";"os/exec";"path/filepath";"syscall")
func main(){
 root,err:=os.MkdirTemp("","starterrors");if err!=nil{panic(err)};defer os.RemoveAll(root)
 denied,format:=filepath.Join(root,"denied"),filepath.Join(root,"format")
 if err:=os.WriteFile(denied,[]byte("#!/bin/sh\nexit 0\n"),0600);err!=nil{panic(err)}
 if err:=os.WriteFile(format,[]byte("not a program\n"),0700);err!=nil{panic(err)}
 for _,item:=range []struct{name,path,dir string}{{"missing",filepath.Join(root,"missing"),""},{"directory",root,""},{"denied",denied,""},{"format",format,""},{"missing-dir",denied,filepath.Join(root,"missing")},{"file-dir",denied,format}}{
  cmd:=exec.Command(item.path);cmd.Dir=item.dir;err:=cmd.Start()
  var pe *os.PathError;ok:=errors.As(err,&pe);op:="";path:=false;if ok{op=pe.Op;path=pe.Path==cmd.Path;if op=="chdir"{path=pe.Path==cmd.Dir}}
  fmt.Printf("%s nil=%t path=%t op=%q missing=%t permission=%t format=%t notdir=%t\n",item.name,err==nil,path,op,errors.Is(err,syscall.ENOENT),errors.Is(err,syscall.EACCES),errors.Is(err,syscall.ENOEXEC),errors.Is(err,syscall.ENOTDIR))
  if err==nil{cmd.Wait()}
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
