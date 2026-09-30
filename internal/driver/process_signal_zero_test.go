package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestProcessSignalZero(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native process queries require macOS or Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/processsignalzero\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"math";"os";"os/exec";"syscall")
func stopped()*os.Process{
 cmd:=exec.Command("sleep","120")
 if err:=cmd.Start();err!=nil{panic(err)}
 defer cmd.Process.Kill()
 fmt.Println("child",cmd.Process.Signal(syscall.Signal(0)))
 if err:=cmd.Process.Kill();err!=nil{panic(err)}
 cmd.Wait()
 return cmd.Process
}
func main(){
 self,err:=os.FindProcess(os.Getpid());if err!=nil{panic(err)}
 fmt.Println("self",self.Signal(syscall.Signal(0)))
 p:=stopped()
 fmt.Println("done",errors.Is(p.Signal(syscall.Signal(0)),os.ErrProcessDone))
 p.Pid=os.Getpid()
 fmt.Println("retained done",errors.Is(p.Signal(syscall.Signal(0)),os.ErrProcessDone))
 absent,err:=os.FindProcess(math.MaxInt32);if err!=nil{panic(err)}
 fmt.Println("absent",errors.Is(absent.Signal(syscall.Signal(0)),os.ErrProcessDone))
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
