package driver_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestGoroutineLaunchTargets(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/plainerror\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import "os"
func main(){
 switch os.Getenv("PANIC_SCENARIO"){
 case "Plain":var f func();go f()
 case "Argument":var f func(int);go f(1)
 case "Result":var f func()int;go f()
 case "Variadic":var f func(...int);go f()
 case "Named":type F func();var f F;go f()
 case "ParentPanic":defer func(){var f func();go f()}();panic("parent")
 case "NonNil":f:=func(){panic("child")};go f()
 case "GenericNil":var f func();launch(f)
 case "ArgumentOrder":var f func(int);go f(func()int{println("argument");return 1}())
 }
 select{}
}
func launch[F ~func()](f F){go f()}
`)
	names := []string{"Plain", "Argument", "Result", "Variadic", "Named", "ParentPanic", "NonNil", "ArgumentOrder", "GenericNil"}
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
			kernel := filepath.Join(out, "main.dill")
			if target == "vm" {
				run(t, out, "dart", "compile", "kernel", "--verbosity=error", "-Ddart.vm.product=false", "bin/main.dart", "-o", kernel)
			}
			for i, name := range names {
				t.Run(name, func(t *testing.T) {
					env := append(os.Environ(), "PANIC_SCENARIO="+name)
					var result programResult
					if target == "vm" {
						result = upstreamOutcome(t, dir, env, runLimit, "dart", "run", "--verbosity=error", kernel)
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
						t.Fatalf("goroutine launch exit=%d\ngot:\n%s\nwant:\n%s", result.Exit, result.Text, wants[name].Text)
					}
				})
			}
		})
	}
}
