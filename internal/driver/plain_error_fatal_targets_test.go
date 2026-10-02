package driver_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestPlainErrorFatalTargets(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/plainerror\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("os";"sync")
func main(){
 var m sync.Mutex
 c:=make(chan int)
 close(c)
 switch os.Getenv("PANIC_SCENARIO"){
 case "FatalSend":defer m.Unlock();c<-1
 case "FatalClose":defer m.Unlock();close(c)
 case "NormalSend":c<-1
 case "NormalClose":close(c)
 case "FatalCloseNil":defer m.Unlock();var n chan int;close(n)
 case "NormalCloseNil":var n chan int;close(n)
 case "FatalInt":defer m.Unlock();panic(7)
 case "FatalNilPointer":defer m.Unlock();var p *int;*p=0
 }
}
`)
	names := []string{"FatalSend", "FatalClose", "NormalSend", "NormalClose", "FatalCloseNil", "NormalCloseNil", "FatalInt", "FatalNilPointer"}
	native := filepath.Join(t.TempDir(), "native")
	run(t, dir, "go", "build", "-o", native, ".")
	wants := map[string]programResult{}
	for _, name := range names {
		result := programOutcomeWithLimitEnv(t, dir, append(os.Environ(), "PANIC_SCENARIO="+name), runLimit, native)
		if result.Exit != 2 {
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
						t.Fatalf("plain-error panic exit=%d\ngot:\n%s\nwant:\n%s", result.Exit, result.Text, wants[name].Text)
					}
				})
			}
		})
	}
}
