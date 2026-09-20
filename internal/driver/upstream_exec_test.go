package driver_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func prepareUpstreamSelfExec(t *testing.T, out, entry string) {
	t.Helper()
	dart, err := exec.LookPath("dart")
	if err != nil {
		t.Fatal(err)
	}
	dart, err = filepath.Abs(dart)
	if err != nil {
		t.Fatal(err)
	}
	probe := filepath.Join(out, "dart_executable.dart")
	writeFile(t, probe, "import 'dart:io';\nvoid main() { print(Platform.resolvedExecutable); }\n")
	dart = strings.TrimSpace(run(t, out, dart, probe))
	launcher := filepath.Join(out, "test executable")
	entry = filepath.Join(out, entry)
	source, err := os.ReadFile(entry)
	if err != nil {
		t.Fatal(err)
	}
	const start = "$boot(() {"
	if strings.Count(string(source), start) != 1 {
		t.Fatal("expected 1 generated test entry point")
	}
	name := strings.ReplaceAll(strconv.Quote(launcher), "$", `\$`)
	// Adapt executable identity in the generated harness, outside upstream assertions.
	code := strings.Replace(string(source), start, "$os.Args[0] = GoString.fromString("+name+");\n"+start, 1)
	writeFile(t, entry, code)
	kernel := filepath.Join(out, "test.dill")
	run(t, out, dart, "compile", "kernel", "-o", kernel, entry)
	quote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'" }
	script := "#!/bin/sh\nexec " + quote(dart) + " " + quote(kernel) + " \"$@\"\n"
	if err := os.WriteFile(launcher, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
}
