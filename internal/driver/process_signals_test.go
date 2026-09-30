package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestProcessSignals(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native process signals require macOS or Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/processsignals\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("bufio";"fmt";"io";"os";"os/exec";"runtime";"strconv";"strings";"syscall";"time")
func stopped(pid int)bool{
 if runtime.GOOS=="linux"{
  b,err:=os.ReadFile("/proc/"+strconv.Itoa(pid)+"/stat");if err!=nil{panic(err)}
  text:=string(b);return strings.HasPrefix(text[strings.LastIndex(text,")")+2:],"T")
 }
 b,err:=exec.Command("ps","-o","stat=","-p",strconv.Itoa(pid)).Output();if err!=nil{panic(err)}
 return strings.HasPrefix(strings.TrimSpace(string(b)),"T")
}
func main(){
 signals:=[]syscall.Signal{syscall.SIGHUP,syscall.SIGINT,syscall.SIGQUIT,syscall.SIGILL,syscall.SIGTRAP,syscall.SIGABRT,syscall.SIGBUS,syscall.SIGFPE,syscall.SIGUSR1,syscall.SIGSEGV,syscall.SIGUSR2,syscall.SIGPIPE,syscall.SIGALRM,syscall.SIGTERM,syscall.SIGCHLD,syscall.SIGCONT,syscall.SIGTSTP,syscall.SIGTTIN,syscall.SIGTTOU,syscall.SIGURG,syscall.SIGXCPU,syscall.SIGXFSZ,syscall.SIGVTALRM,syscall.SIGPROF,syscall.SIGWINCH,syscall.SIGIO,syscall.SIGSYS}
 args:=[]string{"-u","-c","import signal,sys,time\ndef hit(s,f): print(s,flush=True)\nfor n in sys.argv[1:]: signal.signal(int(n),hit)\nsignal.pthread_sigmask(signal.SIG_SETMASK,[])\nprint('ready',flush=True)\nwhile True: time.sleep(1)"}
 for _,s:=range signals{args=append(args,strconv.Itoa(int(s)))}
 cmd:=exec.Command("python3",args...)
 pipe,err:=cmd.StdoutPipe();if err!=nil{panic(err)}
 if err:=cmd.Start();err!=nil{panic(err)}
 reader:=bufio.NewReader(pipe)
 defer func(){cmd.Process.Kill();io.Copy(io.Discard,reader);cmd.Wait()}()
 line:=func()string{
  result:=make(chan string,1)
  go func(){text,err:=reader.ReadString('\n');if err!=nil{text="read error: "+err.Error()};result<-strings.TrimSpace(text)}()
  select{case text:=<-result:return text;case <-time.After(2*time.Second):panic("signal delivery timeout")}
 }
 if text:=line();text!="ready"{panic(text)}
 for _,s:=range signals{
  err:=cmd.Process.Signal(s);delivered:=false
  if err==nil{delivered=line()==strconv.Itoa(int(s))}
  fmt.Println(s,"error",err,"delivered",delivered)
 }
 err=cmd.Process.Signal(syscall.SIGSTOP)
 isStopped:=false
 if err==nil{for i:=0;i<100;i++{if stopped(cmd.Process.Pid){isStopped=true;break};time.Sleep(10*time.Millisecond)}}
 fmt.Println("stop",err,isStopped)
 err=cmd.Process.Signal(syscall.SIGCONT)
 continued:=false;if err==nil{continued=line()==strconv.Itoa(int(syscall.SIGCONT))}
 fmt.Println("continue",err,continued)
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
