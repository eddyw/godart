package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestProcessWaitState(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native process waits require macOS or Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/processwaitstate\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"os";"os/exec";"runtime";"sort";"syscall")
func outcome(p *os.Process,sharedStatus bool)string{
 state,err:=p.Wait()
 if err==nil{return fmt.Sprint("exit ",state.ExitCode())}
 var call *os.SyscallError
 // Linux waiters can reach waitid after another handle consumes the status.
 return fmt.Sprint("error nil=",state==nil," child=",errors.Is(err,syscall.ECHILD)," syscall=",errors.As(err,&call)&&(call.Syscall=="wait"||sharedStatus&&runtime.GOOS=="linux"&&call.Syscall=="waitid"))
}
func run(mode string){
 cmd:=exec.Command("sh","-c","sleep 0.1; exit 7")
 if err:=cmd.Start();err!=nil{panic(err)}
 defer cmd.Wait()
 defer cmd.Process.Kill()
 if mode!="sequential"{
  results:=make(chan string,2)
  other:=cmd.Process
  if mode=="aliases"{var err error;other,err=os.FindProcess(cmd.Process.Pid);if err!=nil{panic(err)}}
  for _,p:=range []*os.Process{cmd.Process,other}{go func(p *os.Process){results<-outcome(p,true)}(p)}
  values:=[]string{<-results,<-results};sort.Strings(values)
  fmt.Println(mode,values)
 }else{
  fmt.Println("first",outcome(cmd.Process,false))
  fmt.Println("second",outcome(cmd.Process,false))
 }
 // An alias can consume the status without marking the original handle done.
 fmt.Println("again",outcome(cmd.Process,mode=="aliases"))
}
func main(){run("sequential");run("concurrent");run("aliases")}
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
