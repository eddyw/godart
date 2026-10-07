package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestExecArgv0(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native argv0 requires macOS or Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/argv0\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"os/exec")
func main(){
 for _, name := range []string{"different", "", string([]byte{255}), "with spaces;$", "-prefix"} {
  cmd:=exec.Command("/bin/sh","-c", "printf '%s' \"$0\"")
  cmd.Args[0]=name
  output,err:=cmd.Output()
  fmt.Printf("%x %v\n",output,err)
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
