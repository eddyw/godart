package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/eddyw/godart/internal/driver"
)

func TestExecWaitClosesPipes(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native process pipes require macOS or Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/execwaitpipes\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("bufio";"fmt";"io";"os/exec")
func main(){
 for _,mode:=range []string{"stdout","stderr","both"}{
  cmd:=exec.Command("sh","-c","printf 'out\\n'; printf 'err\\n' >&2")
  var readers []io.ReadCloser
  if mode!="stderr"{r,err:=cmd.StdoutPipe();if err!=nil{panic(err)};readers=append(readers,r)}
  if mode!="stdout"{r,err:=cmd.StderrPipe();if err!=nil{panic(err)};readers=append(readers,r)}
  if err:=cmd.Start();err!=nil{panic(err)}
  for _,r:=range readers{line,err:=bufio.NewReader(r).ReadString('\n');if err!=nil{panic(err)};fmt.Printf("%s %s",mode,line)}
  fmt.Println(mode,"wait",cmd.Wait())
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
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			var got programResult
			if target == "vm" {
				got = programOutcomeWithLimit(t, dir, 20*time.Second, "dart", filepath.Join(out, "bin/main.dart"))
			} else {
				exe := filepath.Join(out, "pipes.exe")
				run(t, out, "dart", "compile", "exe", filepath.Join(out, "bin/main.dart"), "-o", exe)
				got = programOutcomeWithLimit(t, dir, 20*time.Second, exe)
			}
			if got != want {
				t.Fatalf("Go:\n%s\nGart (exit %d):\n%s", want.Text, got.Exit, got.Text)
			}
		})
	}
}
