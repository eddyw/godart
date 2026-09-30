package driver_test

import (
	"path/filepath"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestProcessRelease(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/processrelease\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"os";"syscall")
func nilRelease()(panics bool){defer func(){panics=recover()!=nil}();var p *os.Process;p.Release();return}
func main(){
 self,err:=os.FindProcess(os.Getpid());if err!=nil{panic(err)}
 for i,p:=range []*os.Process{{},self}{
  fmt.Println(i,"release",p.Release(),"pid",p.Pid)
  fmt.Println(i,"again",p.Release(),"pid",p.Pid)
  fmt.Println(i,"signal",p.Signal(syscall.Signal(0)))
  state,err:=p.Wait()
  fmt.Printf("%d wait nil=%t invalid=%t type=%T\n",i,state==nil,errors.Is(err,syscall.EINVAL),err)
 }
 fmt.Println("nil panic",nilRelease())
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
