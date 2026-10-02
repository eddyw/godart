package driver_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestGoexitRecoveryTargets(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/goexitrecovery\n\ngo 1.27\n")
	names := []string{"GoexitInPanic", "RecoveredPanicAfterGoexit", "RecoverBeforePanicAfterGoexit", "RecoverBeforePanicAfterGoexit2"}
	writeFile(t, filepath.Join(dir, "panic.go"), selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/runtime/testdata/testprog/deadlock.go"), names))
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import ("os";"runtime")
func main(){switch os.Getenv("PANIC_SCENARIO"){case "GoexitInPanic":GoexitInPanic();case "RecoveredPanicAfterGoexit":RecoveredPanicAfterGoexit();case "RecoverBeforePanicAfterGoexit":RecoverBeforePanicAfterGoexit();case "RecoverBeforePanicAfterGoexit2":RecoverBeforePanicAfterGoexit2();case "BlockingRecovered":BlockingRecovered();case "BlockingUnrecovered":BlockingUnrecovered();case "ChildRecovered":ChildRecovered();case "MachineRecovered":MachineRecovered();}}
func BlockingRecovered(){defer func(){runtime.Gosched();recover()}();defer func(){runtime.Gosched();panic("hello")}();runtime.Goexit()}
func BlockingUnrecovered(){defer func(){runtime.Gosched();panic("hello")}();runtime.Goexit()}
func MachineRecovered(){defer func(){recover()}();defer func(){panic("hello")}();runtime.Gosched();runtime.Goexit()}
func ChildRecovered(){c:=make(chan int);go func(){defer func(){c<-1}();RecoverBeforePanicAfterGoexit();println("continued")}();println(<-c)}
`)

	names = append(names, "BlockingRecovered", "BlockingUnrecovered", "ChildRecovered", "MachineRecovered")
	native := filepath.Join(t.TempDir(), "native")
	run(t, dir, "go", "build", "-o", native, ".")
	wants := map[string]programResult{}
	for _, name := range names {
		result := programOutcomeWithLimitEnv(t, dir, append(os.Environ(), "PANIC_SCENARIO="+name), runLimit, native)
		wantExit := 2
		if name == "ChildRecovered" {
			wantExit = 0
		}
		if result.Exit != wantExit {
			t.Fatalf("Go %s: %+v", name, result)
		}
		wants[name] = result
	}
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			for i, name := range names {
				t.Run(name, func(t *testing.T) {
					env := append(os.Environ(), "PANIC_SCENARIO="+name)
					var result programResult
					if target == "vm" {
						result = upstreamOutcome(t, dir, env, runLimit, "dart", filepath.Join(out, "bin/main.dart"))
					} else if i == 0 {
						result = runUpstreamTargetEnv(t, target, out, "bin/main.dart", dir, []string{}, env)
					} else {
						command, args := filepath.Join(out, "tests.exe"), []string{}
						if target == "js" {
							command, args = "node", []string{filepath.Join(out, "run.js")}
						}
						if target == "wasm" {
							command, args = "node", []string{filepath.Join(out, "run.mjs")}
						}
						result = upstreamOutcome(t, dir, env, upstreamLimit(), command, args...)
					}
					if result != wants[name] {
						t.Fatalf("Goexit recovery exit=%d\ngot:\n%s\nwant:\n%s", result.Exit, result.Text, wants[name].Text)
					}
				})
			}
		})
	}
}
