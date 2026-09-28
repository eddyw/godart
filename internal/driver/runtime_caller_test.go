package driver_test

import (
	"go/build"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"testing"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/parity"
)

func TestRuntimeCaller(t *testing.T) {
	dir := filepath.Join(runtime.GOROOT(), "src/runtime")
	fixture, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(fixture, "go.mod"), "module runtime_test\n\ngo 1.27\n")
	writeFile(t, filepath.Join(fixture, "000_package.go"), "package runtime_test\n")
	source := selectedUpstreamSource(t, filepath.Join(dir, "symtab_test.go"), []string{"TestCaller", "TestNilName", "testCallerFoo", "testCallerBar", "TestLineNumber", "lineNumber", "firstLine", "lineVar1", "lineVar2a", "compLit", "arrayLit", "sliceLit", "mapLit", "intLit", "trythis", "recordLines", "l38"})
	writeFile(t, filepath.Join(fixture, "symtab_test.go"), source)
	source = selectedUpstreamSource(t, filepath.Join(dir, "callers_test.go"), []string{"f1", "f2", "f3", "testCallers", "testCallersEqual", "TestCallers", "TestCallersPanic", "TestCallersDoublePanic", "TestCallersAfterRecovery", "TestCallersAbortedPanic", "TestCallersAbortedPanic2", "TestCallersEndlineno", "testNormalEndlineno", "testGenericEndlineno", "testCallerLine", "callerLine"})
	writeFile(t, filepath.Join(fixture, "callers_test.go"), source)
	stats, err := driver.BuildWithStats(driver.Options{Dir: fixture, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TrimPath: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.Binaries) != 1 {
		t.Fatalf("got %d binaries", len(stats.Binaries))
	}
	prepareDart(t, out)
	const filter = "^(TestCaller|TestNilName|TestLineNumber|TestCallers|TestCallersPanic|TestCallersDoublePanic|TestCallersAfterRecovery|TestCallersAbortedPanic|TestCallersAbortedPanic2|TestCallersEndlineno)$"
	args := upstreamTestArgs("-test.run=" + filter)
	result := programOutcome(t, dir, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
	compareUpstream(t, "runtime-caller", "runtime", filter, "vm", dir, result.Text, result.Exit)
	for _, target := range []string{"aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			result := runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			compareUpstream(t, "runtime-caller", "runtime", filter, target, dir, result.Text, result.Exit)
		})
	}
}

func TestRuntimePanicFrames(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/panicframes\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

func stopped(ch chan int) {
	<-ch
	panic("stop")
}

func outer(ch chan int) { stopped(ch) }

func main() {
	ch := make(chan int)
	go func() { ch <- 1 }()
	outer(ch)
}
`)
	native := filepath.Join(t.TempDir(), "native")
	run(t, dir, "go", "build", "-o", native, ".")
	frames := regexp.MustCompile(`(?m)^([^\n]+)\([^\n]*\)\n\s+([^\n]+\.go):(\d+)`)
	positions := func(t *testing.T, command string, args ...string) []string {
		t.Helper()
		var result languageExecution
		text := recordLanguage(t, &result, out, runLimit, command, args...)
		if result.Exit != 2 || result.TimedOut {
			t.Fatalf("panic exit=%d timeout=%t\n%s", result.Exit, result.TimedOut, text)
		}
		var positions []string
		for _, frame := range frames.FindAllStringSubmatch(text, -1) {
			if frame[2] == filepath.Join(dir, "main.go") {
				positions = append(positions, frame[1]+":"+frame[3])
			}
		}
		return positions
	}
	want := positions(t, native)
	if len(want) != 3 {
		t.Fatalf("native panic frame count=%d: %v", len(want), want)
	}
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Format: false}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	t.Run("vm", func(t *testing.T) {
		if got := positions(t, "dart", "bin/main.dart"); !slices.Equal(got, want) {
			t.Fatalf("panic frames: got %v, want %v", got, want)
		}
	})
	t.Run("aot", func(t *testing.T) {
		run(t, out, "dart", "compile", "exe", "bin/main.dart", "-o", "app")
		if got := positions(t, filepath.Join(out, "app")); !slices.Equal(got, want) {
			t.Fatalf("panic frames: got %v, want %v", got, want)
		}
	})
	for _, target := range []string{"js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
			script := "run.js"
			if target == "wasm" {
				script = "run.mjs"
			}
			if got := positions(t, "node", filepath.Join(out, script)); !slices.Equal(got, want) {
				t.Fatalf("panic frames: got %v, want %v", got, want)
			}
		})
	}

}

func TestRuntimePanicLanguage(t *testing.T) {
	inventory, err := parity.DiscoverLanguage(build.Default)
	if err != nil {
		t.Fatal(err)
	}
	var cases []parity.LanguageCase
	for _, fixture := range inventory.Cases {
		if fixture.Name == "fixedbugs/issue5856" {
			cases = append(cases, fixture)
		}
	}
	if len(cases) != 1 {
		t.Fatal("missing upstream panic caller fixture")
	}
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) { runLanguageCases(t, target, inventory, cases) })
	}
}

func TestCallerPrograms(t *testing.T) {
	for _, name := range []string{"callers", "deferpos", "linedir", "inittrace", "paniccallers", "initsuspend"} {
		t.Run(name, func(t *testing.T) {
			dir, err := filepath.Abs(filepath.Join("../../testdata/programs", name))
			if err != nil {
				t.Fatal(err)
			}
			native := filepath.Join(t.TempDir(), "native")
			run(t, dir, "go", "build", "-o", native, ".")
			want := programReference(t, dir, native)
			out := t.TempDir()
			if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Format: true}); err != nil {
				t.Fatal(err)
			}
			prepareDart(t, out)
			for _, target := range []string{"vm", "aot", "js", "wasm"} {
				t.Run(target, func(t *testing.T) {
					var got programResult
					if target == "vm" {
						got = programOutcome(t, out, "dart", "bin/main.dart")
					} else {
						got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
					}
					if got != want {
						t.Fatalf("caller output: got %+v, want %+v", got, want)
					}
				})
			}
		})
	}
}
