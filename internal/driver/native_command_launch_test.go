package driver_test

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestNativeCommandLaunchContracts(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native process launch requires macOS or Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/nativelaunch\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"os";"os/exec";"path/filepath";"strings";"syscall")
func count()int{entries,err:=os.ReadDir("/dev/fd");if err!=nil{panic(err)};return len(entries)}
func main(){
 root,err:=os.MkdirTemp("","nativelaunch");if err!=nil{panic(err)};defer os.RemoveAll(root)
 cmd:=exec.Command("sh","-c","exit 127");err=cmd.Start();fmt.Println("127 start",err==nil,"process",cmd.Process!=nil)
 if err!=nil{panic(err)};err=cmd.Wait();var exit *exec.ExitError;fmt.Println("127 wait",errors.As(err,&exit),"status",cmd.ProcessState.ExitCode(),"format",errors.Is(err,syscall.ENOEXEC))
 script:=filepath.Join(root,"child");if err:=os.WriteFile(script,[]byte("#!/bin/sh\nprintf '%s' \"$1\"\n"),0700);err!=nil{panic(err)}
 cmd=exec.Command("./child","raw-\xff-argument");cmd.Dir=root;output,err:=cmd.Output();fmt.Println("relative",err==nil,"bytes",string(output)=="raw-\xff-argument")
 marker,err:=os.CreateTemp(root,"descriptor-marker-");if err!=nil{panic(err)};defer marker.Close()
 cmd=exec.Command("python3","-c","import os,sys,stat;marker=os.stat(sys.argv[1]);print(','.join(n for n in os.listdir('/dev/fd') if int(n)>2 and os.path.exists('/dev/fd/'+n) and (lambda value:(value.st_dev,value.st_ino)==(marker.st_dev,marker.st_ino) or stat.S_ISFIFO(value.st_mode))(os.fstat(int(n)))))",marker.Name())
 output,err=cmd.Output();fmt.Println("child descriptors",err==nil,strings.TrimSpace(string(output))=="")
 invalid:=filepath.Join(root,"invalid");if err:=os.WriteFile(invalid,[]byte("printf shell-executed\n"),0700);err!=nil{panic(err)}
 before:=count();all:=true
 for i:=0;i<32;i++{cmd:=exec.Command(invalid);err:=cmd.Start();all=all&&errors.Is(err,syscall.ENOEXEC)&&cmd.Process==nil;if err==nil{cmd.Wait()}}
 fmt.Println("format failures",all,"descriptors",count()==before)
 before=count();all=true
 for i:=0;i<32;i++{cmd:=exec.Command("sh","-c","exit 0");err:=cmd.Run();all=all&&err==nil}
 fmt.Println("successful waits",all,"descriptors",count()==before)
}
`)
	want := programOutcome(t, dir, "go", "run", ".")
	if want.Exit != 0 {
		t.Fatalf("native Go failed: %+v", want)
	}
	const expected = "127 start true process true\n127 wait true status 127 format false\nrelative true bytes true\nchild descriptors true true\nformat failures true descriptors true\nsuccessful waits true descriptors true"
	if strings.TrimSpace(want.Text) != expected {
		t.Fatalf("native launch contract: %s", want.Text)
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
