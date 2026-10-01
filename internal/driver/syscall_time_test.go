package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestLinuxSyscallTime(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("syscall.Time requires Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/syscalltime\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"os";"syscall";"time")
func main(){
 if len(os.Args)>1 && os.Args[1]=="unsupported"{
  buf:=[]byte{11,22,33}
  n,err:=syscall.Getcwd(buf)
  if n!=-1 || !errors.Is(err,syscall.ENOSYS) || buf[0]!=11 || buf[1]!=22 || buf[2]!=33 {panic("unsupported Getcwd changed output")}
  value:=syscall.Time_t(17)
  got,err:=syscall.Time(&value)
  if got!=0 || value!=17 || !errors.Is(err,errors.ErrUnsupported){panic("unsupported call changed output")}
  got,err=syscall.Time(nil)
  if got!=0 || !errors.Is(err,errors.ErrUnsupported){panic("nil call must be unsupported")}
  fmt.Println("time unsupported")
  return
 }
 before:=time.Now().Unix()
 got,err:=syscall.Time(nil)
 after:=time.Now().Unix()
 fmt.Println("nil",err,int64(got)>=before&&int64(got)<=after)
 value:=syscall.Time_t(-1)
 before=time.Now().Unix()
 got,err=syscall.Time(&value)
 after=time.Now().Unix()
 fmt.Println("pointer",err,got==value,int64(got)>=before&&int64(got)<=after)
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
			if got.Exit != 0 || got.Text != "time unsupported" {
				t.Fatalf("unsupported target differs: %+v", got)
			}
		})
	}

}
