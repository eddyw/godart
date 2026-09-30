package driver_test

import (
	"path/filepath"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestProcessSignalInitialErrors(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/processsignalerrors\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"os";"syscall")
type unsupported struct{}
func(unsupported) Signal(){}
func(unsupported) String()string{return "unsupported"}
func main(){
 for _,pid:=range []int{0,-1}{
  p:=&os.Process{Pid:pid}
  for _,sig:=range []os.Signal{nil,unsupported{},syscall.Signal(0),os.Interrupt,os.Kill}{
   fmt.Printf("%d %v: %v\n",pid,sig,p.Signal(sig))
  }
  fmt.Printf("%d Kill: %v\n",pid,p.Kill())
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
