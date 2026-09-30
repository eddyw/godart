package driver_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestNilProcessStateStringUpstream(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/nilprocessstate\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "state.go"), "package state\n")
	source := selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/os/os_test.go"), []string{"TestNilProcessStateString"})
	writeFile(t, filepath.Join(dir, "state_test.go"), strings.Replace(source, "package os_test", "package state_test", 1))
	license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.Binaries) != 1 {
		t.Fatalf("got %d test binaries", len(stats.Binaries))
	}
	prepareDart(t, out)
	const filter = "^TestNilProcessStateString$"
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			args := upstreamTestArgs("-test.run=" + filter)
			var got programResult
			if target == "vm" {
				got = upstreamOutcome(t, dir, nil, runLimit, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				got = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "nil-process-state", "os", filter, target, filepath.Join(runtime.GOROOT(), "src/os"), got.Text, got.Exit, true)
		})
	}
}

func TestProcessStateText(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native process signals require macOS or Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/processstatetext\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"os";"os/exec";"strconv";"syscall")
func nilText()(text string){defer func(){if recover()!=nil{text="panic"}}();var p *os.ProcessState;return p.String()}
func main(){
 fmt.Println("nil",nilText())
 fmt.Println("zero",new(os.ProcessState).String())
 for _,sig:=range []syscall.Signal{syscall.SIGHUP,syscall.SIGPIPE,syscall.SIGALRM,syscall.SIGTERM,syscall.SIGKILL,syscall.SIGUSR1,syscall.SIGUSR2}{
  cmd:=exec.Command("python3","-c","import os,signal,sys\ns=int(sys.argv[1])\nif s != signal.SIGKILL: signal.signal(s,signal.SIG_DFL)\nos.kill(os.getpid(),s)",strconv.Itoa(int(sig)))
  err:=cmd.Run()
  if cmd.ProcessState==nil{panic(err)}
  fmt.Println(sig,cmd.ProcessState.String(),err)
 }
 for _,code:=range []string{"0","7","255"}{
  cmd:=exec.Command("sh","-c","exit "+code);err:=cmd.Run()
  if cmd.ProcessState==nil{panic(err)}
  fmt.Println(code,cmd.ProcessState.String(),err)
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
