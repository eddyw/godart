package driver_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestRuntimeRecursivePanicOriginal(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/recursivepanic\n\ngo 1.27\n")
	names := []string{"RecursivePanic", "RecursivePanic2", "RecursivePanic3", "RecursivePanic4", "RecursivePanic5"}
	writeFile(t, filepath.Join(dir, "panic.go"), selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/runtime/testdata/testprog/deadlock.go"), append(append([]string{}, names...), "one", "two", "three")))
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import "os"
func main(){switch os.Getenv("PANIC_SCENARIO"){case "RecursivePanic":RecursivePanic();case "RecursivePanic2":RecursivePanic2();case "RecursivePanic3":RecursivePanic3();case "RecursivePanic4":RecursivePanic4();case "RecursivePanic5":RecursivePanic5()}}
`)
	validator, validatorOut := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(validator, "go.mod"), "module runtime_test\n\ngo 1.27\n")
	writeFile(t, filepath.Join(validator, "000_package.go"), "package runtime_test\n")
	writeFile(t, filepath.Join(validator, "crash_test.go"), selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/runtime/crash_test.go"), []string{"TestRecursivePanic", "TestRecursivePanic2", "TestRecursivePanic3", "TestRecursivePanic4", "TestRecursivePanic5"}))
	writeFile(t, filepath.Join(validator, "adapter_test.go"), `package runtime_test
import ("encoding/json";"os";"testing")
func runTestProg(t *testing.T,binary,name string)string{
 t.Helper()
 var outputs map[string]string
 if err:=json.Unmarshal([]byte(os.Getenv("PANIC_OUTPUTS")),&outputs);err!=nil{t.Fatal(err)}
 return outputs[name]
}
`)
	native := filepath.Join(t.TempDir(), "native")
	run(t, dir, "go", "build", "-o", native, ".")
	observations := map[string]string{}
	for _, name := range names {
		execution := languageExecution{Environment: map[string]string{"PANIC_SCENARIO": name}}
		observations[name] = recordLanguageRaw(t, &execution, dir, runLimit, native)
		if execution.Exit != 2 {
			t.Fatalf("Go %s exit=%d\n%s", name, execution.Exit, observations[name])
		}
	}
	data, _ := json.Marshal(observations)
	const filter = "^(TestRecursivePanic|TestRecursivePanic2|TestRecursivePanic3|TestRecursivePanic4|TestRecursivePanic5)$"
	nativeResult := upstreamOutcome(t, validator, append(os.Environ(), "PANIC_OUTPUTS="+string(data)), runLimit, "go", "test", "-v", "-run", filter, "-count=1")
	if nativeResult.Exit != 0 {
		t.Fatalf("Go original roots: %+v", nativeResult)
	}
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	stats, err := driver.BuildWithStats(driver.Options{Dir: validator, Patterns: []string{"."}, Out: validatorOut, Runtime: runtimeDir(t), Tests: true})
	if err != nil {
		t.Fatal(err)
	}
	prepareDart(t, validatorOut)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			command, args := "dart", []string{filepath.Join(out, "bin/main.dart")}
			if target != "vm" {
				compiled := runUpstreamTargetEnv(t, target, out, "bin/main.dart", dir, []string{}, append(os.Environ(), "PANIC_SCENARIO="+names[0]))
				if compiled.Exit != 2 {
					t.Fatalf("%s first panic: %+v", target, compiled)
				}
				command, args = filepath.Join(out, "tests.exe"), nil
				if target == "js" {
					command, args = "node", []string{filepath.Join(out, "run.js")}
				}
				if target == "wasm" {
					command, args = "node", []string{filepath.Join(out, "run.mjs")}
				}
			}
			for _, name := range names {
				execution := languageExecution{Environment: map[string]string{"PANIC_SCENARIO": name}}
				observations[name] = recordLanguageRaw(t, &execution, dir, upstreamLimit(), command, args...)
				if execution.Exit != 2 {
					t.Fatalf("%s exit=%d\n%s", name, execution.Exit, observations[name])
				}
			}
			data, _ := json.Marshal(observations)
			env := append(os.Environ(), "PANIC_OUTPUTS="+string(data))
			args = upstreamTestArgs("-test.run=" + filter)
			var result programResult
			if target == "vm" {
				result = upstreamOutcome(t, validator, env, runLimit, "dart", append([]string{filepath.Join(validatorOut, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTargetEnv(t, target, validatorOut, stats.Binaries[0].Entry, validator, args, env)
			}
			compareUpstream(t, "runtime-recursive-panic", "runtime", filter, target, filepath.Join(runtime.GOROOT(), "src/runtime"), result.Text, result.Exit, false)
		})
	}
}
