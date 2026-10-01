package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestNativeProcessGroups(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("native process groups require Linux or macOS")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/processgroups\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"os";"syscall")
func main(){
 pid:=syscall.Getpid()
 for _,v:=range [][2]int{{-1,0},{pid,-1},{1<<30,0}}{fmt.Println("invalid",syscall.Setpgid(v[0],v[1]))}
 if os.Args[1]=="session"{
  value,err:=syscall.Setsid();fmt.Println("session",value==pid,err,syscall.Getpgrp()==pid)
  value,err=syscall.Setsid();fmt.Println("repeat",value,err)
  fmt.Println("leader group",syscall.Setpgid(pid,pid))
 }else{
  fmt.Println("group",syscall.Setpgid(pid,pid),syscall.Getpgrp()==pid)
  value,err:=syscall.Setsid();fmt.Println("leader session",value,err)
  wrapped:=int(int64(pid)+(1<<32));fmt.Println("wrapped group",syscall.Setpgid(wrapped,wrapped),syscall.Getpgrp()==pid)
 }
}
`)
	native := filepath.Join(dir, "native")
	run(t, dir, "go", "build", "-o", native, ".")
	want := map[string]programResult{}
	for _, mode := range []string{"session", "group"} {
		want[mode] = programOutcome(t, dir, native, mode)
		if want[mode].Exit != 0 {
			t.Fatalf("native %s: %+v", mode, want[mode])
		}
	}
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			executable := filepath.Join(out, "groups")
			if target == "aot" {
				run(t, out, "dart", "compile", "exe", "bin/main.dart", "-o", executable)
			}
			for _, mode := range []string{"session", "group"} {
				t.Run(mode, func(t *testing.T) {
					var got programResult
					if target == "vm" {
						got = programOutcome(t, dir, "dart", filepath.Join(out, "bin/main.dart"), mode)
					} else {
						got = programOutcome(t, dir, executable, mode)
					}
					if got != want[mode] {
						t.Fatalf("%s differs\nGo: %+v\nGart: %+v", mode, want[mode], got)
					}
				})
			}
		})
	}
}
