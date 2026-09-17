package driver_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/eddyw/godart/internal/driver"
)

// TestPrograms compiles each Go program of testdata and runs the Dart output.
// The output must be equal to the output of the Go program.
func TestPrograms(t *testing.T) {
	root, err := filepath.Abs("../../testdata/programs")
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := filepath.Abs("../../runtime/goruntime")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	selected := testShard(t, names)
	for _, entry := range entries {
		if !entry.IsDir() || !selected[entry.Name()] {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			t.Parallel()
			dir := filepath.Join(root, entry.Name())
			native := filepath.Join(t.TempDir(), "program")
			run(t, dir, "go", "build", "-o", native, ".")
			want := programOutcome(t, dir, native)
			out := t.TempDir()
			err := driver.Build(driver.Options{
				Trace:    traceBuild(t),
				Dir:      dir,
				Patterns: []string{"./..."},
				Out:      out,
				Runtime:  runtime,
				Format:   false,
			})
			if err != nil {
				t.Fatalf("the build failed:\n%v", err)
			}
			prepareDart(t, out)
			got := programOutcome(t, out, "dart", "--packages=.dart_tool/package_config.json", "bin/main.dart")
			if got != want {
				t.Errorf("the output or exit status differs\nGo:\n%+v\nDart:\n%+v", want, got)
			}
		})
	}
}

// runProgram runs a compiled program. A panic writes a stack trace, and the 2
// runtimes write a different trace, thus the test cuts the trace away.
func runProgram(t *testing.T, dir string, name string, args ...string) string {
	t.Helper()
	result := programOutcome(t, dir, name, args...)
	if result.Exit != 0 {
		t.Fatalf("%s %v: exit %d\n%s", name, args, result.Exit, result.Text)
	}
	return result.Text
}

type programResult struct {
	Text string
	Exit int
}

func programOutcome(t *testing.T, dir, name string, args ...string) programResult {
	t.Helper()
	return programOutcomeWithLimit(t, dir, runLimit, name, args...)
}

func programOutcomeWithLimit(t *testing.T, dir string, limit time.Duration, name string, args ...string) programResult {
	t.Helper()
	start := time.Now()
	text, err, ok := output(dir, nil, limit, name, args...)
	t.Logf("command=%s %v duration=%.3fs", name, args, time.Since(start).Seconds())
	if !ok {
		t.Fatalf("%s %v exceeded %v:\n%s", name, args, limit, text)
	}
	code := 0
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			code = e.ExitCode()
		} else {
			t.Fatal(err)
		}
	}
	return programResult{cutTrace(string(text)), code}
}

func cutTrace(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		// Go reports the signal of a crash, and the trace follows. The Dart
		// VM prints the Dart trace of its own. The blank line in front of
		// each trace takes the trim.
		if strings.HasPrefix(line, "goroutine ") || strings.HasPrefix(line, "[signal ") {
			return strings.TrimRight(strings.Join(lines[:i], "\n"), "\n")
		}
	}
	return strings.TrimRight(text, "\n")
}

// TestAOT proves that the output also compiles ahead of time. The Dart VM and
// AOT are the 2 targets of the project.
func TestAOT(t *testing.T) {
	source, err := filepath.Abs("../../testdata/programs/multi")
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := filepath.Abs("../../runtime/goruntime")
	if err != nil {
		t.Fatal(err)
	}
	want := runProgram(t, source, "go", "run", ".")
	out := t.TempDir()
	if err := driver.Build(driver.Options{
		Trace:    traceBuild(t),
		Dir:      source,
		Patterns: []string{"./..."},
		Out:      out,
		Runtime:  runtime,
		Format:   true,
	}); err != nil {
		t.Fatal(err)
	}
	run(t, out, "dart", "pub", "get")
	run(t, out, "dart", "compile", "exe", "bin/main.dart", "-o", "app")
	if got := runProgram(t, out, filepath.Join(out, "app")); got != want {
		t.Errorf("the output of the AOT build differs\nGo:\n%s\nDart:\n%s", want, got)
	}
}

// TestWasm proves the minimal web target. A pure program compiles with dart
// compile wasm and runs under node, and the output stays equal to the output
// of go run. A program that reaches dart:io fails in that compile, per the
// web decision of 2026-09-01.
func TestWasm(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skipf("the machine holds no node: %v", err)
	}
	source, err := filepath.Abs("../../testdata/programs/hello")
	if err != nil {
		t.Fatal(err)
	}
	runtimeDir, err := filepath.Abs("../../runtime/goruntime")
	if err != nil {
		t.Fatal(err)
	}
	want := runProgram(t, source, "go", "run", ".")
	out := t.TempDir()
	if err := driver.Build(driver.Options{
		Trace:    traceBuild(t),
		Dir:      source,
		Patterns: []string{"./..."},
		Out:      out,
		Runtime:  runtimeDir,
		Format:   true,
	}); err != nil {
		t.Fatal(err)
	}
	run(t, out, "dart", "pub", "get")
	run(t, out, "dart", "compile", "wasm", "bin/main.dart", "-o", "main.wasm")
	runner := `import { compile } from './main.mjs';
import { readFile } from 'node:fs/promises';

const bytes = await readFile(new URL('./main.wasm', import.meta.url));
const app = await compile(bytes);
const instance = await app.instantiate({});
instance.invokeMain();
`
	if err := os.WriteFile(filepath.Join(out, "run.mjs"), []byte(runner), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := runProgram(t, out, "node", "run.mjs"); got != want {
		t.Errorf("the output of the wasm build differs\nGo:\n%s\nnode:\n%s", want, got)
	}
}

// TestJS proves the JS target on 1 program of 64-bit integer work. dart2js
// maps a plain int to a JS number of 53 bits, thus the emitted code rides
// the pair form of goint_js.dart there. The run needs node.
func TestJS(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skipf("the machine holds no node: %v", err)
	}
	for _, name := range []string{"intwrap", "reflectint64", "genmixed", "genops", "genericvalue", "nanoclock", "globalptr", "structconv", "reflectblock", "uniquecompat"} {
		t.Run(name, func(t *testing.T) {
			source, err := filepath.Abs(filepath.Join("../../testdata/programs", name))
			if err != nil {
				t.Fatal(err)
			}
			runtimeDir, err := filepath.Abs("../../runtime/goruntime")
			if err != nil {
				t.Fatal(err)
			}
			want := runProgram(t, source, "go", "run", ".")
			out := t.TempDir()
			if err := driver.Build(driver.Options{
				Trace:    traceBuild(t),
				Dir:      source,
				Patterns: []string{"./..."},
				Out:      out,
				Runtime:  runtimeDir,
				Format:   true,
			}); err != nil {
				t.Fatal(err)
			}
			run(t, out, "dart", "pub", "get")
			run(t, out, "dart", "compile", "js", "bin/main.dart", "-o", "main.js")
			if got := runProgram(t, out, "node", "main.js"); got != want {
				t.Errorf("the output of the JS build differs\nGo:\n%s\nnode:\n%s", want, got)
			}
		})
	}
}

// TestZoneinfoZip proves the zip reader of the time overlay. The variable
// ZONEINFO points at the zone zip file of the Go tree, thus the 2 runs read
// the same zip, and the output must stay equal.
func TestZoneinfoZip(t *testing.T) {
	source, err := filepath.Abs("../../testdata/programs/tzuse")
	if err != nil {
		t.Fatal(err)
	}
	runtimeDir, err := filepath.Abs("../../runtime/goruntime")
	if err != nil {
		t.Fatal(err)
	}
	zip := filepath.Join(strings.TrimSpace(run(t, source, "go", "env", "GOROOT")),
		"lib", "time", "zoneinfo.zip")
	if _, err := os.Stat(zip); err != nil {
		t.Skipf("the Go tree holds no zoneinfo.zip: %v", err)
	}
	env := append(os.Environ(), "ZONEINFO="+zip)
	want := runProgramEnv(t, source, env, "go", "run", ".")
	out := t.TempDir()
	if err := driver.Build(driver.Options{
		Trace:    traceBuild(t),
		Dir:      source,
		Patterns: []string{"./..."},
		Out:      out,
		Runtime:  runtimeDir,
		Format:   true,
	}); err != nil {
		t.Fatal(err)
	}
	run(t, out, "dart", "pub", "get")
	got := runProgramEnv(t, out, env, "dart", "run", "bin/main.dart")
	if got != want {
		t.Errorf("the output differs under ZONEINFO\nGo:\n%s\nDart:\n%s", want, got)
	}
}

// runProgramEnv runs a compiled program with an environment.
func runProgramEnv(t *testing.T, dir string, env []string, name string, args ...string) string {
	t.Helper()
	start := time.Now()
	text, err, ok := output(dir, env, runLimit, name, args...)
	t.Logf("command=%s %v duration=%.3fs", name, args, time.Since(start).Seconds())
	if !ok {
		t.Fatalf("%s %v did not end inside %v in %s", name, args, runLimit, dir)
	}
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, text)
	}
	return cutTrace(string(text))
}

func run(t *testing.T, dir string, name string, args ...string) string {
	t.Helper()
	start := time.Now()
	text, err, ok := output(dir, nil, runLimit, name, args...)
	t.Logf("command=%s %v duration=%.3fs", name, args, time.Since(start).Seconds())
	if !ok {
		t.Fatalf("%s %v did not end inside %v in %s", name, args, runLimit, dir)
	}
	if err != nil {
		t.Fatalf("%s %v failed in %s: %v\n%s", name, args, dir, err, text)
	}
	return string(text)
}

// runNodeJS runs a dart2js build under node and gives its output.
//
// dart2js schedules a microtask through self, which a browser and a worker
// each hold and node does not. The shim below stands in for it.
func runNodeJS(t *testing.T, dir string) string {
	t.Helper()
	const shim = `globalThis.self = globalThis;
require('./main.js');
`
	if err := os.WriteFile(filepath.Join(dir, "run.js"), []byte(shim), 0o644); err != nil {
		t.Fatal(err)
	}
	return runProgram(t, dir, "node", "run.js")
}

// TestWebIO proves that the whole io surface compiles for the web. The Go
// package os names each function of the io shim, thus a program that only
// reads an environment value once failed in the Dart compile. Each web form
// now gives an honest cause, and the program runs. Read decision 251.
func TestWebIO(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skipf("the machine holds no node: %v", err)
	}
	source, err := filepath.Abs("../../testdata/web/webio")
	if err != nil {
		t.Fatal(err)
	}
	runtimeDir, err := filepath.Abs("../../runtime/goruntime")
	if err != nil {
		t.Fatal(err)
	}
	const want = `goos: true
pid: 0
open: open /etc/hosts: function not implemented
mkdir: mkdir /x: function not implemented
get: an error`
	out := t.TempDir()
	if err := driver.Build(driver.Options{
		Trace:    traceBuild(t),
		Dir:      source,
		Patterns: []string{"./..."},
		Out:      out,
		Runtime:  runtimeDir,
		Format:   true,
	}); err != nil {
		t.Fatal(err)
	}
	run(t, out, "dart", "pub", "get")

	run(t, out, "dart", "compile", "js", "bin/main.dart", "-o", "main.js")
	if got := runNodeJS(t, out); got != want {
		t.Errorf("the output of the JS build differs\nwant:\n%s\ngot:\n%s", want, got)
	}

	run(t, out, "dart", "compile", "wasm", "bin/main.dart", "-o", "main.wasm")
	runner := `import { compile } from './main.mjs';
import { readFile } from 'node:fs/promises';

const bytes = await readFile(new URL('./main.wasm', import.meta.url));
const app = await compile(bytes);
const instance = await app.instantiate({});
instance.invokeMain();
`
	if err := os.WriteFile(filepath.Join(out, "run.mjs"), []byte(runner), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := runProgram(t, out, "node", "run.mjs"); got != want {
		t.Errorf("the output of the wasm build differs\nwant:\n%s\ngot:\n%s", want, got)
	}
}
