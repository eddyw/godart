package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDarwinProcessObservations(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Process observations require macOS")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/accept\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"syscall")
func must(err error){if err!=nil{panic(err)}}
func main(){
 var original syscall.Rlimit;must(syscall.Getrlimit(syscall.RLIMIT_NOFILE,&original));defer func(){must(syscall.Setrlimit(syscall.RLIMIT_NOFILE,&original))}()
 initial:=syscall.Getdtablesize();fmt.Println("initial table",initial>0)
 fmt.Println("initial taint",syscall.Issetugid())
 for _,size:=range []uint64{64,128,96}{value:=original;value.Cur=size;must(syscall.Setrlimit(syscall.RLIMIT_NOFILE,&value));var actual syscall.Rlimit;must(syscall.Getrlimit(syscall.RLIMIT_NOFILE,&actual));fmt.Println("table",size,syscall.Getdtablesize(),actual.Cur==size,value.Cur==size);fmt.Println("taint",syscall.Issetugid())}
 must(syscall.Setrlimit(syscall.RLIMIT_NOFILE,&original));fmt.Println("restored table",syscall.Getdtablesize()==initial)
}
`)
	want := runProgram(t, dir, "go", "run", ".")
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			var got programResult
			if target == "vm" {
				got = upstreamOutcome(t, dir, nil, runLimit, "dart", filepath.Join(out, "bin/main.dart"))
			} else {
				got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
			}
			if got.Exit != 0 || got.Text != want {
				t.Fatalf("%s: exit %d\nGo:\n%s\nGart:\n%s", target, got.Exit, want, got.Text)
			}
		})
	}
}
