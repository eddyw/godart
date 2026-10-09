package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDarwinSync(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Sync requires macOS")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/accept\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"os";"path/filepath";"syscall")
func must(err error){if err!=nil{panic(err)}}
func main(){
 dir,err:=os.MkdirTemp("","syscall-sync");must(err);defer os.RemoveAll(dir)
 path:=filepath.Join(dir,"file");fd,err:=syscall.Open(path,syscall.O_CREAT|syscall.O_RDWR|syscall.O_TRUNC,0600);must(err);duplicate,err:=syscall.Dup(fd);must(err)
 n,err:=syscall.Write(fd,[]byte{0,255,17,19});must(err);must(syscall.Sync());position,err:=syscall.Seek(duplicate,0,1);must(err);fmt.Println("first",n,position);must(syscall.Close(fd));n,err=syscall.Write(duplicate,[]byte{23,29});must(err);must(syscall.Sync());position,err=syscall.Seek(duplicate,0,1);must(err);fmt.Println("duplicate",n,position);_,err=syscall.Seek(duplicate,0,0);must(err);data:=[]byte{31,37,41,43,47,53,59,61,67,71};n,err=syscall.Read(duplicate,data[2:8]);must(err);fmt.Println("bytes",n,data);must(syscall.Close(duplicate));must(syscall.Sync());data,err=os.ReadFile(path);must(err);fmt.Println("retained",data)
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
