package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCommandEnvironment(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native child environments require macOS or Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/cmdenv\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"os";"os/exec";"strings";"syscall")
func main(){
 python,err:=exec.LookPath("python3");if err!=nil{panic(err)}
 root,err:=os.MkdirTemp("","cmdenv");if err!=nil{panic(err)};defer os.RemoveAll(root)
 if err:=os.Setenv("PWD","parent-marker");err!=nil{panic(err)}
 for _,explicit:=range []bool{false,true}{
  cmd:=exec.Command(python,"-c","import os;print(os.environ.get('PWD','missing'));print(os.environ.get('DUPE','missing'));print(os.environ.get('','missing'))")
  cmd.Dir=root;want:=root+"\nmissing\nmissing"
  if explicit{cmd.Env=[]string{"PWD=explicit","DUPE=first","DUPE=last","=blank"};want="explicit\nlast\nblank"}
  out,err:=cmd.Output();if err!=nil{panic(err)}
  fmt.Println("explicit",explicit,"environment",strings.TrimSpace(string(out))==want)
 }
 for _,entry:=range []string{"BAD=one\x00two","BAD\x00=one"}{
  cmd:=exec.Command(python,"-c","pass");cmd.Env=[]string{entry}
  err:=cmd.Start()
  fmt.Printf("nul error=%v process=%t\n",err,cmd.Process!=nil)
  if err==nil{cmd.Wait()}
 }
 for _,kind:=range []string{"path","arg0","arg1","dir","dir-explicit"}{
  cmd:=exec.Command(python,"-c","pass")
  switch kind{case "path":cmd.Path+="\x00suffix";case "arg0":cmd.Args[0]+="\x00suffix";case "arg1":cmd.Args=append(cmd.Args,"one\x00two");case "dir":cmd.Dir=root+"\x00suffix";case "dir-explicit":cmd.Dir=root+"\x00suffix";cmd.Env=[]string{}}
  err:=cmd.Start();var pe *os.PathError
  ok:=errors.As(err,&pe)
  fields:=ok&&pe.Op=="fork/exec"&&pe.Path==cmd.Path
  if kind=="dir-explicit"{fields=ok&&pe.Op=="chdir"&&pe.Path==cmd.Dir}
  fmt.Println("input",kind,"invalid",errors.Is(err,syscall.EINVAL),"patherror",fields,"environment",err!=nil&&err.Error()=="exec: environment variable contains NUL","process",cmd.Process!=nil)
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
