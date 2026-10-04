package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNativeFileLocks(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("Native file locks require Linux or macOS")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/flock\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
 "fmt"
 "os"
 "path/filepath"
 "syscall"
)

func errline(label string,err error) {
 code,typed:=err.(syscall.Errno)
 fmt.Println(label,code,typed)
}

func main() {
 dir,err:=os.MkdirTemp("","flock-contract")
 if err!=nil { panic(err) }
 defer os.RemoveAll(dir)
 path:=filepath.Join(dir,"locked")
 first,err:=syscall.Open(path,syscall.O_CREAT|syscall.O_RDWR,0600)
 if err!=nil { panic(err) }
 second,err:=syscall.Open(path,syscall.O_RDWR,0)
 if err!=nil { panic(err) }
 errline("exclusive",syscall.Flock(first,syscall.LOCK_EX|syscall.LOCK_NB))
 errline("repeat",syscall.Flock(first,syscall.LOCK_EX|syscall.LOCK_NB))
 errline("other shared conflict",syscall.Flock(second,syscall.LOCK_SH|syscall.LOCK_NB))
 errline("other exclusive conflict",syscall.Flock(second,syscall.LOCK_EX|syscall.LOCK_NB))
 duplicate,err:=syscall.Dup(first)
 if err!=nil { panic(err) }
 errline("first close",syscall.Close(first))
 errline("duplicate retains lock",syscall.Flock(second,syscall.LOCK_EX|syscall.LOCK_NB))
 errline("duplicate unlock",syscall.Flock(duplicate,syscall.LOCK_UN))
 errline("second shared",syscall.Flock(second,syscall.LOCK_SH|syscall.LOCK_NB))
 errline("duplicate shared",syscall.Flock(duplicate,syscall.LOCK_SH|syscall.LOCK_NB))
 errline("upgrade conflict",syscall.Flock(second,syscall.LOCK_EX|syscall.LOCK_NB))
 errline("duplicate unlock again",syscall.Flock(duplicate,syscall.LOCK_UN))
 errline("upgrade",syscall.Flock(second,syscall.LOCK_EX|syscall.LOCK_NB))
 errline("invalid operation",syscall.Flock(second,128))
 errline("invalid retains lock",syscall.Flock(duplicate,syscall.LOCK_EX|syscall.LOCK_NB))
 n,err:=syscall.Write(second,[]byte{0,128,255})
 fmt.Println("lock retains write",n)
 errline("write",err)
 errline("second close",syscall.Close(second))
 errline("close releases lock",syscall.Flock(duplicate,syscall.LOCK_EX|syscall.LOCK_NB))
 errline("unlock",syscall.Flock(duplicate,syscall.LOCK_UN))
 errline("unlock unlocked",syscall.Flock(duplicate,syscall.LOCK_UN))
 errline("duplicate close",syscall.Close(duplicate))
 errline("invalid descriptor",syscall.Flock(-1,syscall.LOCK_EX|syscall.LOCK_NB))
 data,err:=os.ReadFile(path)
 if err!=nil { panic(err) }
 fmt.Println("actual file",data)
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
