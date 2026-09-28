package driver_test

import (
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestRuntimePanicGoroutine(t *testing.T) {
	source, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(source, "go.mod"), "module example.com/panicgoroutine\n\ngo 1.27\n")
	writeFile(t, filepath.Join(source, "main.go"), `package main
import ("os"; "runtime"; "strings"; "time")
func fail() {
 buf := make([]byte, 1024)
 n := runtime.Stack(buf, false)
 println("origin:", strings.Fields(string(buf[:n]))[1])
 panic("worker failure")
}
func worker() {
 if os.Args[1] == "resumed" { time.Sleep(time.Millisecond) }
 if os.Args[1] == "nested" {
  defer fail()
  panic("first failure")
 }
 fail()
}
func main() {
 if os.Args[1] == "main" { worker(); return }
 go worker()
 select {}
}
`)
	run(t, source, "go", "build", "-o", "native", ".")
	if err := driver.Build(driver.Options{Dir: source, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	run(t, out, "dart", "compile", "exe", "bin/main.dart", "-o", "app")
	origin := regexp.MustCompile(`(?m)^origin: ([0-9]+)$`)
	header := regexp.MustCompile(`(?m)^goroutine ([0-9]+) \[running\]:$`)
	for _, scenario := range []string{"main", "worker", "resumed", "nested"} {
		for _, target := range []string{"native", "vm", "aot"} {
			t.Run(scenario+"/"+target, func(t *testing.T) {
				command, args := filepath.Join(source, "native"), []string{scenario}
				if target == "vm" {
					command, args = "dart", []string{filepath.Join(out, "bin/main.dart"), scenario}
				} else if target == "aot" {
					command = filepath.Join(out, "app")
				}
				var result languageExecution
				text := recordLanguage(t, &result, source, runLimit, command, args...)
				want, got := origin.FindStringSubmatch(text), header.FindStringSubmatch(text)
				if result.Exit != 2 || result.TimedOut || len(want) != 2 || len(got) != 2 || got[1] != want[1] {
					t.Fatalf("exit=%d: panic header must match runtime.Stack identity\n%s", result.Exit, text)
				}
				if (scenario == "main") != (got[1] == "1") {
					t.Fatalf("unexpected %s goroutine identity %s", scenario, got[1])
				}
			})
		}
	}
}

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
			args = upstreamTestArgs("-test.run=^TestPanicTraceback$")
			if target == "vm" {
				result = programOutcome(t, validator, "dart", append([]string{filepath.Join(validatorOut, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, validatorOut, stats.Binaries[0].Entry, validator, args)
			}
			compareUpstream(t, "runtime-panic", "runtime", "^TestPanicTraceback$", target, filepath.Join(runtime.GOROOT(), "src/runtime"), result.Text, result.Exit, false)
		})
	}
}
