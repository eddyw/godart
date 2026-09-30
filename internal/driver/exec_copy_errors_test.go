package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestCommandCopyErrors(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native child pipes require macOS or Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/copyerrors\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("context";"errors";"fmt";"io";"os/exec";"time")
var failure=errors.New("copy failure")
type badReader struct{}
func(badReader)Read(p []byte)(int,error){return 0,failure}
type badWriter struct{short bool}
func(w badWriter)Write(p []byte)(int,error){if w.short{return 0,nil};return 0,failure}
func main(){
 for _,kind:=range []string{"stdin","stdout","stderr","short"}{
  for _,code:=range []string{"0","7"}{
   cmd:=exec.Command("sh","-c","cat >/dev/null; printf data; printf error >&2; exit "+code)
   switch kind{case "stdin":cmd.Stdin=badReader{};case "stdout":cmd.Stdout=badWriter{};case "stderr":cmd.Stderr=badWriter{};case "short":cmd.Stdout=badWriter{true}}
   err:=cmd.Run();var exit *exec.ExitError
   fmt.Println(kind,code,"copy",errors.Is(err,failure),"short",errors.Is(err,io.ErrShortWrite),"exit",errors.As(err,&exit))
  }
 }
 for _,fd:=range []string{"1","2"}{
  ctx,cancel:=context.WithTimeout(context.Background(),5*time.Second)
  cmd:=exec.CommandContext(ctx,"python3","-c","import os,sys\ntry:\n while True: os.write(int(sys.argv[1]),b'x'*65536)\nexcept BrokenPipeError: sys.exit(7)",fd)
  if fd=="1"{cmd.Stdout=badWriter{}}else{cmd.Stderr=badWriter{}}
  err:=cmd.Run();var exit *exec.ExitError
  code:=-1;if errors.As(err,&exit){code=exit.ExitCode()}
  fmt.Println("closed",fd,"exit",code,"timeout",ctx.Err()!=nil)
  cancel()
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
