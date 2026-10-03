package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestNativeSocketDescriptors(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("Pipe validation requires native descriptors")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/pipebackpressure\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("errors";"fmt";"os";"syscall")
func main(){
 for _, family := range []int{syscall.AF_INET, syscall.AF_INET6, syscall.AF_UNIX} {
  fd,err:=syscall.Socket(family,syscall.SOCK_STREAM,0)
  if err!=nil{panic(err)}
  file:=os.NewFile(uintptr(fd),"native-socket")
  info,err:=file.Stat();if err!=nil{panic(err)}
  fmt.Println("socket", info.Mode()&os.ModeSocket!=0)
  if err:=file.Close();err!=nil{panic(err)}
  fmt.Println("ownership",errors.Is(syscall.Close(fd),syscall.EBADF))
 }
 fd,err:=syscall.Socket(-1,syscall.SOCK_STREAM,0)
 fmt.Println("invalid",fd==-1,err!=nil)
 syscall.SocketDisableIPv6=true
 fd,err=syscall.Socket(syscall.AF_INET6,syscall.SOCK_STREAM,0)
 fmt.Println("ipv6 disabled",fd==-1,errors.Is(err,syscall.EAFNOSUPPORT))
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
