package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestCommandOutputPipeClose(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native output pipes require macOS or Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/pipeclose\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("context";"errors";"fmt";"io";"os";"os/exec";"time")
func report(op string,err error){var pe *os.PathError;ok:=errors.As(err,&pe);fmt.Println(op,"closed",errors.Is(err,os.ErrClosed),"path",ok&&pe.Op==op&&pe.Path=="|0")}
func main(){
 for _,fd:=range []string{"1","2"}{
  ctx,cancel:=context.WithTimeout(context.Background(),5*time.Second)
  cmd:=exec.CommandContext(ctx,"python3","-c","import os,sys\nfd=int(sys.argv[1]);os.write(fd,b'R')\nsys.stdin.buffer.read(1)\ntry:\n while True: os.write(fd,b'x'*65536)\nexcept BrokenPipeError: sys.exit(7)",fd)
  input,err:=cmd.StdinPipe();if err!=nil{panic(err)}
  var output io.ReadCloser
  if fd=="1"{output,err=cmd.StdoutPipe()}else{output,err=cmd.StderrPipe()};if err!=nil{panic(err)}
  if err=cmd.Start();err!=nil{panic(err)}
  b:=make([]byte,1);if _,err=io.ReadFull(output,b);err!=nil{panic(err)}
  n,err:=output.Read(nil);fmt.Println("zero",n,err)
  reading:=make(chan error,1)
  go func(){_,err:=output.Read(make([]byte,1));reading<-err}()
  time.Sleep(10*time.Millisecond)
  fmt.Println("pipe",fd,"ready",string(b),"close",output.Close())
  report("read",<-reading)
  _,err=output.Read(b);report("read",err)
  report("close",output.Close())
  input.Write([]byte("X"));input.Close()
  err=cmd.Wait();var exit *exec.ExitError;code:=-1;if errors.As(err,&exit){code=exit.ExitCode()}
  fmt.Println("exit",code,"timeout",ctx.Err()!=nil)
  _,err=output.Read(b);report("read",err)
  cancel()
 }
 cmd:=exec.Command("sh","-c","printf x")
 output,err:=cmd.StdoutPipe();if err!=nil{panic(err)}
 if err=cmd.Start();err!=nil{panic(err)}
 if _,err=io.ReadAll(output);err!=nil{panic(err)}
 if err=cmd.Wait();err!=nil{panic(err)}
 _,err=output.Read(make([]byte,1));report("read",err)
 report("close",output.Close())
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
