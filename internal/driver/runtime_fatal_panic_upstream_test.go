package driver_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestRuntimeFatalPanic(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/fatalpanic\n\ngo 1.27\n")
	source := filepath.Join(runtime.GOROOT(), "src/runtime/testdata/testprog/panicprint.go")
	writeFile(t, filepath.Join(dir, "panic.go"), selectedUpstreamSource(t, source, nil))
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import "os"
var programs = map[string]func(){}
func register(name string, f func()) { programs[name] = f }
func main() { programs[os.Getenv("PANIC_SCENARIO")]() }
`)
	validator, validatorOut := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(validator, "go.mod"), "module runtime_test\n\ngo 1.27\n")
	writeFile(t, filepath.Join(validator, "000_package.go"), "package runtime_test\n")
	root := selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/runtime/panic_test.go"), []string{"TestPanicWithDirectlyPrintableCustomTypes"})
	writeFile(t, filepath.Join(validator, "panic_test.go"), root)
	names := regexp.MustCompile(`\{"(panic\w+)"`).FindAllStringSubmatch(root, -1)
	if len(names) != 19 {
		t.Fatalf("got %d original cases", len(names))
	}
	writeFile(t, filepath.Join(validator, "adapter_test.go"), `package runtime_test
import ("encoding/json"; "os"; "testing")
func runTestProg(t *testing.T, binary, name string) string {
 t.Helper()
 var outputs map[string]string
 if err := json.Unmarshal([]byte(os.Getenv("PANIC_OUTPUTS")), &outputs); err != nil { t.Fatal(err) }
 return outputs[name]
}
`)
	native := filepath.Join(t.TempDir(), "native")
	run(t, dir, "go", "build", "-o", native, ".")
	observations := map[string]string{}
	for _, name := range names {
		result := programOutcomeWithLimitEnv(t, dir, append(os.Environ(), "PANIC_SCENARIO="+name[1]), runLimit, native)
		if result.Exit != 2 {
			t.Fatalf("Go panic %s: %+v", name[1], result)
		}
		observations[name[1]] = result.Text
	}
	nativeOutputs, _ := json.Marshal(observations)
	nativeResult := upstreamOutcome(t, validator, append(os.Environ(), "PANIC_OUTPUTS="+string(nativeOutputs)), runLimit, "go", "test", "-v", "-run", "^TestPanicWithDirectlyPrintableCustomTypes$", "-count=1")
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
			kernel := filepath.Join(out, "main.dill")
			if target == "vm" {
				run(t, out, "dart", "compile", "kernel", "--verbosity=error", "-Ddart.vm.product=false", "bin/main.dart", "-o", kernel)
			}
			for i, name := range names {
				env := append(os.Environ(), "PANIC_SCENARIO="+name[1])
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
				if result.Exit != 2 {
					t.Fatalf("%s exit=%d\n%s", name[1], result.Exit, result.Text)
				}
				observations[name[1]] = result.Text
			}
			data, _ := json.Marshal(observations)
			env := append(os.Environ(), "PANIC_OUTPUTS="+string(data))
			args := upstreamTestArgs("-test.run=^TestPanicWithDirectlyPrintableCustomTypes$")
			var result programResult
			if target == "vm" {
				result = upstreamOutcome(t, validator, env, runLimit, "dart", append([]string{filepath.Join(validatorOut, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTargetEnv(t, target, validatorOut, stats.Binaries[0].Entry, validator, args, env)
			}
			compareUpstream(t, "runtime-fatal-panic", "runtime", "^TestPanicWithDirectlyPrintableCustomTypes$", target, filepath.Join(runtime.GOROOT(), "src/runtime"), result.Text, result.Exit, false)
		})
	}
}
