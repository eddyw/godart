package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCommandStderrCapture(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native stderr capture requires macOS or Linux")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/stderrcapture\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("bytes";"crypto/sha256";"errors";"fmt";"os/exec";"strconv")
func main(){
 for _,n:=range []int{0,32767,32768,32769,65536,65537,123456,1000000}{
  for _,explicit:=range []bool{false,true}{
   cmd:=exec.Command("python3","-c","import os,sys\nn=int(sys.argv[1]);p=(bytes(range(251))*((n+250)//251))[:n]\nfor i in range(0,n,997): os.write(2,p[i:i+997])\nos.write(1,b'done');sys.exit(7)",strconv.Itoa(n))
   var stderr bytes.Buffer;if explicit{cmd.Stderr=&stderr}
   out,err:=cmd.Output();var exit *exec.ExitError;if !errors.As(err,&exit){panic(err)}
   fmt.Printf("%d explicit=%t out=%q captured=%d:%x supplied=%d:%x\n",n,explicit,out,len(exit.Stderr),sha256.Sum256(exit.Stderr),stderr.Len(),sha256.Sum256(stderr.Bytes()))
  }
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

func TestCommandStderrSaverUpstream(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/stderrsaver\n\ngo 1.27\n")
	source := selectedUpstreamSource(t, "../overlay/std/os/exec/capture.go", []string{"prefixSuffixSaver", "prefixSuffixSaver.Write", "prefixSuffixSaver.fill", "prefixSuffixSaver.Bytes"})
	source = strings.Replace(source, "//go:build dart\n", "", 1)
	writeFile(t, filepath.Join(dir, "capture.go"), source)
	source = selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/os/exec/internal_test.go"), []string{"TestPrefixSuffixSaver"})
	writeFile(t, filepath.Join(dir, "capture_test.go"), source)
	license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
	t.Log(run(t, dir, "go", "test", "-v", "."))
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true})
	if err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			args := upstreamTestArgs("-test.run=^TestPrefixSuffixSaver$")
			var got programResult
			if target == "vm" {
				got = upstreamOutcome(t, dir, nil, runLimit, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				got = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "exec-stderr-saver", "os/exec", "^TestPrefixSuffixSaver$", target, filepath.Join(runtime.GOROOT(), "src/os/exec"), got.Text, got.Exit, true)
		})
	}
}
