package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestProcessPriority(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("process priority requires Linux or macOS")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/processpriority\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"runtime";"syscall")
func code(err error) int {if err==nil{return 0};var value syscall.Errno;if !errors.As(err,&value){panic(err)};return int(value)}
func must(err error){if err!=nil{panic(err)}}
func main(){
 pid:=syscall.Getpid()
 initial,err:=syscall.Getpriority(syscall.PRIO_PROCESS,pid);must(err)
 wrapped:=int(int64(pid)+(int64(1)<<32))
 for i,c:=range [][2]int{{syscall.PRIO_PROCESS,pid},{syscall.PRIO_PROCESS,wrapped},{-1,pid},{syscall.PRIO_PROCESS,-1},{syscall.PRIO_PROCESS,1<<30}}{
  value,err:=syscall.Getpriority(c[0],c[1]);fmt.Println("query",i,value,code(err))
 }
 nice:=initial
 if runtime.GOOS=="linux"{nice=20-initial}
 target:=nice
 if target<10{target=10}
 must(syscall.Setpriority(syscall.PRIO_PROCESS,pid,target))
 value,err:=syscall.Getpriority(syscall.PRIO_PROCESS,pid);must(err)
 expected:=target
 if runtime.GOOS=="linux"{expected=20-target}
 if value!=expected{panic("actual process priority differs")}
 fmt.Println("updated",value)
 must(syscall.Setpriority(syscall.PRIO_PROCESS,wrapped,target))
 for i,c:=range [][2]int{{-1,pid},{syscall.PRIO_PROCESS,-1},{syscall.PRIO_PROCESS,1<<30}}{
  fmt.Println("set error",i,code(syscall.Setpriority(c[0],c[1],target)))
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
}
