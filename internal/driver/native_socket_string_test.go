package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNativeSocketStringOptions(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("Raw socket options require native descriptors")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/socketstring\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"syscall")
func main(){
 fmt.Println("invalid empty descriptor",errors.Is(syscall.SetsockoptString(-1,0,0,""),syscall.EBADF))
 fd,err:=syscall.Socket(syscall.AF_INET,syscall.SOCK_DGRAM,0);if err!=nil{panic(err)}
 for i,value:=range []string{"","\x01\x01\x00\x00","\xff\x00\x00\x00",""} {
  err:=syscall.SetsockoptString(fd,syscall.IPPROTO_IP,syscall.IP_OPTIONS,value)
  fmt.Println("option",i,"success",err==nil,"invalid",errors.Is(err,syscall.EINVAL))
 }
 fmt.Println("unknown option",syscall.SetsockoptString(fd,-1,0,"\x00\xff")!=nil)
 fmt.Println("ownership",syscall.Close(fd)==nil)
 fmt.Println("closed descriptor",errors.Is(syscall.SetsockoptString(fd,syscall.IPPROTO_IP,syscall.IP_OPTIONS,""),syscall.EBADF))
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
