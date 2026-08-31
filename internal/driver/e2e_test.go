package driver_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

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
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			t.Parallel()
			dir := filepath.Join(root, entry.Name())
			want := runProgram(t, dir, "go", "run", ".")
			out := t.TempDir()
			err := driver.Build(driver.Options{
				Dir:      dir,
				Patterns: []string{"./..."},
				Out:      out,
				Runtime:  runtime,
				Format:   true,
			})
			if err != nil {
				t.Fatalf("the build failed:\n%v", err)
			}
			run(t, out, "dart", "pub", "get")
			got := runProgram(t, out, "dart", "run", "bin/main.dart")
			if got != want {
				t.Errorf("the output differs\nGo:\n%s\nDart:\n%s", want, got)
			}
		})
	}
}

// runProgram runs a compiled program. A panic writes a stack trace, and the 2
// runtimes write a different trace, thus the test cuts the trace away.
func runProgram(t *testing.T, dir string, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	text, _ := cmd.CombinedOutput()
	return cutTrace(string(text))
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
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = env
	text, _ := cmd.CombinedOutput()
	return cutTrace(string(text))
}

func run(t *testing.T, dir string, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	text, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed in %s: %v\n%s", name, args, dir, err, text)
	}
	return string(text)
}
