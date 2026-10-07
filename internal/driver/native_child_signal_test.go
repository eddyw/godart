package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNativeChildSignalDispositions(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native signal dispositions require macOS or Linux")
	}
	dir, out, childDir := t.TempDir(), t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(childDir, "child.c"), `#include <signal.h>
#include <stdio.h>
int main(void) {
 struct sigaction pipe,ttou;
 if(sigaction(SIGPIPE,NULL,&pipe)!=0 || sigaction(SIGTTOU,NULL,&ttou)!=0)return 1;
 printf("pipe %d ttou %d\n",pipe.sa_handler==SIG_IGN,ttou.sa_handler==SIG_IGN);
 return 0;
}
`)
	child := filepath.Join(childDir, "child")
	run(t, childDir, "cc", "-Wall", "-Wextra", "-Werror", "child.c", "-o", child)
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/childsignals\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"os";"os/exec";"os/signal";"syscall")
func child(){output,err:=exec.Command(os.Args[1]).Output();fmt.Printf("%s%v %v\n",output,err,signal.Ignored(syscall.SIGPIPE))}
func main(){child();signal.Ignore(syscall.SIGPIPE);child();signal.Reset(syscall.SIGPIPE);child();c:=make(chan os.Signal,1);signal.Notify(c,syscall.SIGPIPE);child();signal.Stop(c);child()}
`)
	want := programOutcome(t, dir, "go", "run", ".", child)
	if want.Exit != 0 {
		t.Fatalf("native Go failed: %+v", want)
	}
	t.Log("native child signal dispositions", want.Text)
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			var got programResult
			if target == "vm" {
				got = programOutcome(t, dir, "dart", filepath.Join(out, "bin/main.dart"), child)
			} else {
				got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{child})
			}
			if got != want {
				t.Fatalf("Go:\n%s\nGart (exit %d):\n%s", want.Text, got.Exit, got.Text)
			}
		})
	}
}
