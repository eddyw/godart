package driver_test

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

type nativeImage struct {
	Command []string `json:"command"`
	SHA256  string   `json:"sha256"`
}

// upstreamOutcome retains process failures for the parity report.
// Exit 124 indicates a timeout. Exit 127 indicates a command error without
// an exit status.
func upstreamOutcome(t *testing.T, dir string, env []string, limit time.Duration, name string, args ...string) programResult {
	t.Helper()
	start := time.Now()
	text, err, completed := output(dir, env, limit, name, args...)
	t.Logf("command=%s %v duration=%.3fs", name, args, time.Since(start).Seconds())
	if !completed {
		return programResult{string(text), 124}
	}
	code := 0
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			code = e.ExitCode()
		} else {
			return programResult{fmt.Sprintf("%s\ncommand %s: %v", text, name, err), 127}
		}
	}
	return programResult{cutTrace(string(text)), code}
}

func prepareUpstreamNativeImage(t *testing.T, dir, out, entry string) nativeImage {
	t.Helper()
	name := filepath.Join(out, "native.test")
	args := []string{"test", "-c", "-o", name, "."}
	run(t, dir, "go", args...)
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	result := nativeImage{Command: append([]string{"go"}, args...), SHA256: fmt.Sprintf("%x", sha256.Sum256(data))}
	entry = filepath.Join(out, entry)
	source, err := os.ReadFile(entry)
	if err != nil {
		t.Fatal(err)
	}
	const start = "_main();"
	if strings.Count(string(source), start) != 1 {
		t.Fatal("expected 1 generated test entry point")
	}
	imports := upstreamOSImport(t, out)
	quoted := strings.ReplaceAll(strconv.Quote(name), "$", `\$`)
	writeFile(t, entry, imports+strings.Replace(string(source), start,
		"$upstreamOS.Args[0] = GoString.fromString("+quoted+");\n"+start, 1))
	return result
}

func prepareUpstreamSelfExec(t *testing.T, out, entry string) string {
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
	const start = "_main();"
	if strings.Count(string(source), start) != 1 {
		t.Fatal("expected 1 generated test entry point")
	}
	name := strings.ReplaceAll(strconv.Quote(launcher), "$", `\$`)
	// Adapt executable identity in the generated harness, outside upstream assertions.
	imports := upstreamOSImport(t, out)
	code := imports + strings.Replace(string(source), start, "$upstreamOS.Args[0] = GoString.fromString("+name+");\n"+start, 1)
	writeFile(t, entry, code)
	executableSource := filepath.Join(out, "lib", "std", "os", "extra.part.dart")
	source, err = os.ReadFile(executableSource)
	if err != nil {
		t.Fatal(err)
	}
	const executable = "$executable()"
	if strings.Count(string(source), executable) != 1 {
		t.Fatal("expected 1 generated executable query")
	}
	writeFile(t, executableSource, strings.Replace(string(source), executable,
		"(GoString.fromString("+name+"), GoString.empty)", 1))
	kernel := filepath.Join(out, "test_executable.dill")
	run(t, out, "dart", "run", "--verbosity=error", "goruntime:build_assets")
	run(t, out, "dart", "compile", "kernel", "-Ddart.vm.product=false", "-o", kernel, entry)
	sdk := filepath.Dir(filepath.Dir(dart))
	assets := filepath.Join(out, "test_native_assets.dill")
	run(t, out, filepath.Join(sdk, "bin", "dartaotruntime"),
		filepath.Join(sdk, "bin", "snapshots", "frontend_server_aot.dart.snapshot"),
		"--sdk-root", sdk, "--platform", filepath.Join(sdk, "lib", "_internal", "vm_platform_strong.dill"),
		"--target=vm", "--native-assets-only",
		"--native-assets="+filepath.Join(out, ".dart_tool", "native_assets.yaml"),
		"--output-dill", assets, entry)
	kernelData, err := os.ReadFile(kernel)
	if err != nil {
		t.Fatal(err)
	}
	assetData, err := os.ReadFile(assets)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(kernel, append(kernelData, assetData...), 0600); err != nil {
		t.Fatal(err)
	}
	quote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'" }
	script := "#!/bin/sh\nexec " + quote(dart) + " " + quote(kernel) + " \"$@\"\n"
	if err := os.WriteFile(launcher, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	return launcher
}

func upstreamOSImport(t *testing.T, out string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(out, "pubspec.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	line, _, _ := strings.Cut(string(data), "\n")
	name := strings.TrimPrefix(line, "name: ")
	// A package URI keeps the import in the same Dart library instance.
	return "import " + strconv.Quote("package:"+name+"/std/os.dart") + " as $upstreamOS;\n"
}
