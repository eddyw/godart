package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestSyscallPlatformDeclarations(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native syscall declarations require macOS or Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/syscallconstants\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import ("errors"; "fmt"; "io/fs"; "os"; "syscall")
var read [syscall.SYS_READ+1]byte
var stop [syscall.SIGSTOP]byte
var refused [syscall.ECONNREFUSED]byte
func main() {
 fmt.Println(os.O_RDONLY,os.O_WRONLY,os.O_RDWR,os.O_APPEND,os.O_CREATE,os.O_EXCL,os.O_TRUNC,os.O_SYNC,os.SEEK_SET,os.SEEK_CUR,os.SEEK_END)
 fmt.Println(len(stop),len(refused),syscall.SIGCONT,syscall.AF_INET,syscall.AF_INET6)
 fmt.Println(len(read),syscall.SYS_READ,syscall.SYS_WRITE,syscall.SYS_CLOSE,syscall.SYS_EXIT,syscall.SYS_FCNTL)
 for i:=0;i<256;i++ {
  e:=syscall.Errno(i)
  fmt.Printf("%d %q temporary=%t timeout=%t permission=%t exist=%t notexist=%t unsupported=%t\n",i,e.Error(),e.Temporary(),e.Timeout(),errors.Is(e,fs.ErrPermission),errors.Is(e,fs.ErrExist),errors.Is(e,fs.ErrNotExist),errors.Is(e,errors.ErrUnsupported))
 }
 fmt.Println(syscall.Errno(^uintptr(0)))
 for i:=-1;i<=65;i++ {fmt.Printf("signal %d %q\n",i,syscall.Signal(i).String())}
}
`)
	want := runProgram(t, dir, "go", "run", ".")
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	got := runProgram(t, dir, "dart", filepath.Join(out, "bin/main.dart"))
	if got != want {
		t.Fatalf("syscall declarations differ:\nGo:\n%s\nGart:\n%s", want, got)
	}
	for _, target := range []string{"aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			got := runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
			if got.Exit != 0 || got.Text != want {
				t.Fatalf("%s syscall declarations differ:\nGo:\n%s\nGart (%d):\n%s", target, want, got.Exit, got.Text)
			}
		})
	}
}
