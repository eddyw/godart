package driver_test

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/eddyw/godart/internal/driver"
)

func TestFindProcessKill(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native process signals require macOS or Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/processkill\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"os";"os/exec";"strconv";"syscall")
func main(){
 pid,err:=strconv.Atoi(os.Args[1]);if err!=nil{panic(err)}
 p,err:=os.FindProcess(pid);if err!=nil{panic(err)}
 fmt.Println("kill",p.Kill())
 child:=exec.Command("sleep","120")
 if err:=child.Start();err!=nil{panic(err)}
 defer child.Process.Kill()
 fmt.Println("child kill",child.Process.Kill())
 err=child.Wait()
 var exit *exec.ExitError
 fmt.Println("child exit",errors.As(err,&exit),child.ProcessState.Sys().(syscall.WaitStatus).Signal()==syscall.SIGKILL)
 fmt.Println("child done",errors.Is(child.Process.Kill(),os.ErrProcessDone))
}
`)
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"go", "vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			child := exec.Command("sleep", "120")
			if err := child.Start(); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() { done <- child.Wait() }()
			t.Cleanup(func() { _ = child.Process.Kill() })
			args := []string{strconv.Itoa(child.Process.Pid)}
			var got programResult
			switch target {
			case "go":
				got = programOutcome(t, dir, "go", "run", ".", args[0])
			case "vm":
				got = programOutcome(t, dir, "dart", filepath.Join(out, "bin/main.dart"), args[0])
			case "aot":
				got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, args)
			}
			if got.Exit != 0 || got.Text != "kill <nil>\nchild kill <nil>\nchild exit true true\nchild done true" {
				t.Fatalf("kill result: %+v", got)
			}
			select {
			case <-done:
				status, ok := child.ProcessState.Sys().(syscall.WaitStatus)
				if !ok || !status.Signaled() || status.Signal() != syscall.SIGKILL {
					t.Fatalf("child status: %v", child.ProcessState)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("child remains alive after Kill")
			}
		})
	}
}
