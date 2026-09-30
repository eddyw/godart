package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestCommandInputPipeClose(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native input pipes require macOS or Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/inputclose\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("bytes";"errors";"fmt";"os";"os/exec")
func report(op string,err error){var pe *os.PathError;ok:=errors.As(err,&pe);fmt.Println(op,"closed",errors.Is(err,os.ErrClosed),"path",ok&&pe.Op==op&&pe.Path=="|1")}
func main(){
 cmd:=exec.Command("cat");var out bytes.Buffer;cmd.Stdout=&out
 input,err:=cmd.StdinPipe();if err!=nil{panic(err)}
 if err=cmd.Start();err!=nil{panic(err)}
 n,err:=input.Write([]byte("data"));fmt.Println("write",n,err)
 fmt.Println("first close",input.Close())
 report("close",input.Close())
 for _,p:=range [][]byte{nil,[]byte("x")}{n,err=input.Write(p);fmt.Println("count",n);report("write",err)}
 fmt.Println("wait",cmd.Wait(),"output",out.String())
 report("close",input.Close())
 cmd=exec.Command("sh","-c","exit 0");input,err=cmd.StdinPipe();if err!=nil{panic(err)}
 if err=cmd.Start();err!=nil{panic(err)}
 fmt.Println("wait open",cmd.Wait())
 report("close",input.Close())
 n,err=input.Write([]byte("x"));fmt.Println("count",n);report("write",err)
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
				got = upstreamOutcome(t, dir, nil, 20*time.Second, "dart", filepath.Join(out, "bin/main.dart"))
			} else {
				got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
			}
			if got != want {
				t.Fatalf("Go:\n%s\nGart (exit %d):\n%s", want.Text, got.Exit, got.Text)
			}
		})
	}
}
