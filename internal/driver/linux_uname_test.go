package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestLinuxUname(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Uname requires Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/uname\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"os";"syscall")
func fill(u *syscall.Utsname){
 for _,field:=range []*[65]int8{&u.Sysname,&u.Nodename,&u.Release,&u.Version,&u.Machine,&u.Domainname}{for i:=range field{field[i]=-91}}
}
func main(){
 var box struct{Before uint64;Value syscall.Utsname;After uint64}
 box.Before=0x0123456789abcdef;box.After=0xfedcba9876543210;fill(&box.Value)
 if len(os.Args)>1&&os.Args[1]=="unsupported"{
  want:=box.Value
  if !errors.Is(syscall.Uname(&box.Value),errors.ErrUnsupported)||box.Value!=want{panic("Uname changed output or did not return unsupported")}
  if !errors.Is(syscall.Uname(nil),errors.ErrUnsupported){panic("nil Uname must return unsupported")}
  fmt.Println("uname unsupported");return
 }
 fmt.Println("nil",syscall.Uname(nil))
 err:=syscall.Uname(&box.Value);fmt.Println("query",err);if err!=nil{panic(err)}
 fmt.Println("bounds",box.Before==0x0123456789abcdef,box.After==0xfedcba9876543210)
 for i,field:=range [][65]int8{box.Value.Sysname,box.Value.Nodename,box.Value.Release,box.Value.Version,box.Value.Machine,box.Value.Domainname}{fmt.Println("field",i,field)}
 first:=box.Value;fill(&box.Value);fmt.Println("repeat",syscall.Uname(&box.Value),box.Value==first)
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
	for _, target := range []string{"js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			got := runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{"unsupported"})
			if got.Exit != 0 || got.Text != "uname unsupported" {
				t.Fatalf("unsupported target differs: %+v", got)
			}
		})
	}
}
