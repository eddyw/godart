package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestDarwinSysctl(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Sysctl requires macOS")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/sysctl\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"os";"syscall")
func main(){
 if len(os.Args)>1&&os.Args[1]=="unsupported"{
  text,err:=syscall.Sysctl("kern.ostype")
  if text!=""||!errors.Is(err,errors.ErrUnsupported){panic("string query must be unsupported")}
  value,err:=syscall.SysctlUint32("hw.ncpu")
  if value!=0||!errors.Is(err,errors.ErrUnsupported){panic("uint32 query must be unsupported")}
  fmt.Println("sysctl unsupported");return
 }
 for i,name:=range []string{"kern.ostype","kern.osrelease","hw.ncpu","hw.byteorder","hw.memsize","kern.safeboot","gart.missing","","kern.ostype\x00suffix","kern.ostype\x00",".kern.ostype","kern.ostype.",string([]byte{0xff})}{
  text,err:=syscall.Sysctl(name);fmt.Printf("string %d %x %v\n",i,text,err)
  value,err:=syscall.SysctlUint32(name);fmt.Println("uint32",i,value,err)
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
	targets := []string{"vm", "aot"}
	for _, target := range targets {
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
	for _, target := range []string{"js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			got := runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{"unsupported"})
			if got.Exit != 0 || got.Text != "sysctl unsupported" {
				t.Fatalf("unsupported target differs: %+v", got)
			}
		})
	}

}
