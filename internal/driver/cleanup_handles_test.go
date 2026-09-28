package driver_test

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCleanupHandles(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skipf("node is unavailable: %v", err)
	}
	out := t.TempDir()
	writeFile(t, filepath.Join(out, "pubspec.yaml"), fmt.Sprintf("name: cleanup_handles\nenvironment:\n  sdk: ^3.13.0\ndependencies:\n  goruntime:\n    path: %q\n", runtimeDir(t)))
	writeFile(t, filepath.Join(out, "main.dart"), `import 'package:goruntime/goruntime.dart' as rt;

void cleanup() {}

void main() {
  final target = rt.Iface(rt.$ptrDesc(rt.$typeInt), rt.Box(rt.$gi(1)));
  for (var i = 0; i < 10000000; i++) {
    rt.$stopCleanup(rt.$addCleanup(target, null, false, cleanup));
  }
  print('completed');
  print(target.value != null);
}
`)
	prepareDart(t, out)
	run(t, out, "dart", "compile", "js", "main.dart", "-o", "main.js")
	writeFile(t, filepath.Join(out, "run.js"), "globalThis.self = globalThis; require('./main.js');\n")
	got := programOutcome(t, out, "node", "--max-old-space-size=64", "run.js")
	if got.Exit != 0 || got.Text != "completed\ntrue" {
		t.Fatalf("cleanup handle storage: %+v", got)
	}
}
