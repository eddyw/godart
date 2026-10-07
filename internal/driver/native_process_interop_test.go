package driver_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNativeProcessInterop(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native child status requires macOS or Linux")
	}
	out := t.TempDir()
	fixture, err := os.ReadFile(filepath.Join(runtimeDir(t), "test/fixtures/native_process_status.dart"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(out, "pubspec.yaml"), fmt.Sprintf("name: native_process_status\nenvironment:\n  sdk: ^3.13.0\ndependencies:\n  goruntime:\n    path: %s\n", runtimeDir(t)))
	writeFile(t, filepath.Join(out, "main.dart"), string(fixture))
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			var got programResult
			if target == "vm" {
				got = programOutcome(t, out, "dart", filepath.Join(out, "main.dart"))
			} else {
				got = runUpstreamTarget(t, target, out, "main.dart", out, []string{})
			}
			if got.Exit != 0 || strings.TrimSpace(got.Text) != "native process statuses pass" {
				t.Fatalf("native process status (exit %d): %s", got.Exit, got.Text)
			}
		})
	}
}
