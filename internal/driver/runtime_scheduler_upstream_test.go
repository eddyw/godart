package driver_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestRuntimeSchedulerOriginal(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/scheduleroriginal\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "misc.go"), selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/runtime/testdata/testprog/misc.go"), []string{"NumGoroutine"}))
	writeFile(t, filepath.Join(dir, "main.go"), "package main\nfunc main(){NumGoroutine()}\n")
	validator, validatorOut := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(validator, "go.mod"), "module runtime_test\n\ngo 1.27\n")
	writeFile(t, filepath.Join(validator, "000_package.go"), "package runtime_test\n")
	writeFile(t, filepath.Join(validator, "proc_test.go"), selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/runtime/proc_test.go"), []string{"TestYieldProgress", "testYieldProgress", "TestNumGoroutine"}))
	writeFile(t, filepath.Join(validator, "adapter_test.go"), `package runtime_test
import ("os";"testing")
func runTestProg(t *testing.T,binary,name string)string{
 t.Helper()
 return os.Getenv("PANIC_OUTPUT")
}
`)
	native := filepath.Join(t.TempDir(), "native")
	run(t, dir, "go", "build", "-o", native, ".")
	execution := languageExecution{}
	observation := recordLanguageRaw(t, &execution, dir, runLimit, native)
	if execution.Exit != 0 {
		t.Fatalf("Go exit=%d\n%s", execution.Exit, observation)
	}
	const filter = "^(TestYieldProgress|TestNumGoroutine)$"
	nativeResult := upstreamOutcome(t, validator, append(os.Environ(), "PANIC_OUTPUT="+observation), runLimit, "go", "test", "-v", "-run", filter, "-count=1")
	if nativeResult.Exit != 0 {
		t.Fatalf("Go original root: %+v", nativeResult)
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
				compiled := runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
				if compiled.Exit != 0 {
					t.Fatalf("%s programme: %+v", target, compiled)
				}
				command, args = filepath.Join(out, "tests.exe"), nil
				if target == "js" {
					command, args = "node", []string{filepath.Join(out, "run.js")}
				}
				if target == "wasm" {
					command, args = "node", []string{filepath.Join(out, "run.mjs")}
				}
			}
			execution := languageExecution{}
			observation := recordLanguageRaw(t, &execution, dir, upstreamLimit(), command, args...)
			if execution.Exit != 0 {
				t.Fatalf("%s exit=%d\n%s", target, execution.Exit, observation)
			}
			env := append(os.Environ(), "PANIC_OUTPUT="+observation)
			args = upstreamTestArgs("-test.run=" + filter)
			var result programResult
			if target == "vm" {
				result = upstreamOutcome(t, validator, env, runLimit, "dart", append([]string{filepath.Join(validatorOut, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTargetEnv(t, target, validatorOut, stats.Binaries[0].Entry, validator, args, env)
			}
			compareUpstream(t, "runtime-scheduler", "runtime", filter, target, filepath.Join(runtime.GOROOT(), "src/runtime"), result.Text, result.Exit, false)
		})
	}
}
