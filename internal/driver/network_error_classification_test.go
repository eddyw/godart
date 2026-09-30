package driver_test

import (
	"path/filepath"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestNetworkErrorClassification(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/networkerror\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"net";"os";"syscall")
type classified struct{ timeout, temporary bool }
func (e classified) Error() string { return "classified" }
func (e classified) Timeout() bool { return e.timeout }
func (e classified) Temporary() bool { return e.temporary }
func main(){
 causes:=[]error{nil,errors.New("plain"),syscall.EAGAIN,syscall.ETIMEDOUT,syscall.EINTR,syscall.EMFILE,syscall.ENFILE,syscall.ECONNRESET,syscall.ECONNABORTED,syscall.EINVAL,classified{true,false},classified{false,true}}
 for _,op:=range []string{"accept","read","write"} {
  for i,cause:=range causes {
   for j,wrapped:=range []error{cause,&os.SyscallError{Syscall:op,Err:cause},fmt.Errorf("wrapped: %w",cause),&os.SyscallError{Syscall:op,Err:&os.SyscallError{Syscall:op,Err:cause}}} {
    e:=&net.OpError{Op:op,Net:"tcp",Err:wrapped}
    fmt.Printf("%s/%d/%d timeout=%t temporary=%t\n",op,i,j,e.Timeout(),e.Temporary())
   }
  }
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
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			var result programResult
			if target == "vm" {
				result = programOutcome(t, dir, "dart", filepath.Join(out, "bin/main.dart"))
			} else {
				result = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
			}
			if result != want {
				t.Fatalf("Go:\n%s\nGart (exit %d):\n%s", want.Text, result.Exit, result.Text)
			}
		})
	}
}
