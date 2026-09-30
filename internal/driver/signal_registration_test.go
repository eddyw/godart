package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestSignalRegistrations(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native signals require macOS or Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/signalregistrations\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"os";"os/signal";"syscall";"time")
func main(){
 s:=syscall.SIGWINCH
 for _,name:=range []string{"explicit-all","all-explicit","ignore-explicit","ignore-all","reset-all","ignore-empty"}{
  signal.Reset()
  a,b:=make(chan os.Signal,4),make(chan os.Signal,4)
  switch name {
  case "explicit-all": signal.Notify(a,s);signal.Notify(a)
  case "all-explicit": signal.Notify(a);signal.Notify(a,s)
  case "ignore-explicit": signal.Notify(a,s);signal.Ignore(s);signal.Notify(b,s)
  case "ignore-all": signal.Ignore(s);signal.Notify(b)
  case "reset-all": signal.Notify(a);signal.Reset(s);signal.Notify(b,s)
  case "ignore-empty": signal.Ignore();fmt.Println(name,"ignored",signal.Ignored(s));signal.Notify(b,s)
  }
  if err:=syscall.Kill(os.Getpid(),s);err!=nil{panic(err)}
  time.Sleep(200*time.Millisecond)
  fmt.Println(name,"a",len(a),"b",len(b),"ignored",signal.Ignored(s))
  signal.Stop(a);signal.Stop(b)
 }
 signal.Reset()
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
