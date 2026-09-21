package driver_test

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestRuntimePanicTraceback(t *testing.T) {
	source := filepath.Join(runtime.GOROOT(), "src/runtime/testdata/testprog/deadlock.go")
	fixture, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(fixture, "go.mod"), "module example.com/panictrace\n\ngo 1.27\n")
	writeFile(t, filepath.Join(fixture, "panic.go"), selectedUpstreamSource(t, source, []string{"PanicTraceback", "pt1", "pt2"}))
	writeFile(t, filepath.Join(fixture, "main.go"), "package main\nfunc main() { PanicTraceback() }\n")
	if err := driver.Build(driver.Options{Dir: fixture, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	validator, validatorOut := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(validator, "go.mod"), "module runtime_test\n\ngo 1.27\n")
	writeFile(t, filepath.Join(validator, "000_package.go"), "package runtime_test\n")
	writeFile(t, filepath.Join(validator, "crash_test.go"), selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/runtime/crash_test.go"), []string{"TestPanicTraceback"}))
	outputPath := filepath.Join(validator, "panic-output.txt")
	writeFile(t, filepath.Join(validator, "adapter_test.go"), fmt.Sprintf(`package runtime_test
import ("os"; "testing")
func runTestProg(t *testing.T, binary, name string) string {
 t.Helper()
 if binary != "testprog" || name != "PanicTraceback" { t.Fatal("unexpected panic fixture") }
 data, err := os.ReadFile(%q)
 if err != nil { t.Fatal(err) }
 return string(data)
}
`, outputPath))
	stats, err := driver.BuildWithStats(driver.Options{Dir: validator, Patterns: []string{"."}, Out: validatorOut, Runtime: runtimeDir(t), Tests: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.Binaries) != 1 {
		t.Fatalf("got %d test binaries", len(stats.Binaries))
	}
	prepareDart(t, validatorOut)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			command, args := "dart", []string{"bin/main.dart"}
			if target == "aot" {
				run(t, out, "dart", "compile", "exe", "bin/main.dart", "-o", "app")
				command, args = filepath.Join(out, "app"), nil
			}
			var execution languageExecution
			text := recordLanguage(t, &execution, out, runLimit, command, args...)
			if execution.Exit != 2 || execution.TimedOut {
				t.Fatalf("panic exit=%d timeout=%t\n%s", execution.Exit, execution.TimedOut, text)
			}
			writeFile(t, outputPath, text)
			var result programResult
			args = []string{"-test.v", "-test.run=^TestPanicTraceback$"}
			if target == "vm" {
				result = programOutcome(t, validator, "dart", append([]string{filepath.Join(validatorOut, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, validatorOut, stats.Binaries[0].Entry, validator, args)
			}
			compareUpstream(t, "runtime-panic", "runtime", "^TestPanicTraceback$", target, filepath.Join(runtime.GOROOT(), "src/runtime"), result.Text, result.Exit)
		})
	}
}
