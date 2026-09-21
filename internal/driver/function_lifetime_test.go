package driver_test

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestFunctionMetadataLifetime(t *testing.T) {
	out := t.TempDir()
	writeFile(t, filepath.Join(out, "pubspec.yaml"), fmt.Sprintf("name: function_metadata\nenvironment:\n  sdk: ^3.13.0\ndependencies:\n  goruntime:\n    path: %q\n", runtimeDir(t)))
	writeFile(t, filepath.Join(out, "main.dart"), `import 'dart:async';
import 'package:goruntime/goruntime.dart';
class Payload { final int value; Payload(this.value); }
(WeakReference<Payload>, int) create(int value) {
 final payload = Payload(value);
 final function = $functionValue(() => payload,
   GoString.fromString('fixture.capture|capture.go|7'));
 if (function().value != value) throw StateError('lost live capture');
 final pc = $functionPC(function);
 if (pc == null) throw StateError('missing metadata');
 return (WeakReference(payload), pc);
}
void pressure() {
 final garbage = <Object>[];
 for (var i = 0; i < 20000; i++) { garbage.add(List<int>.filled(64, i)); }
 garbage.clear();
}
Future<void> main() async {
 final (first, firstPC) = create(1);
 final (second, secondPC) = create(2);
 if (firstPC != secondPC) throw StateError('different code identities');
 final deadline = DateTime.now().add(const Duration(seconds: 4));
 while (true) {
  await Future<void>.delayed(const Duration(milliseconds: 10));
  pressure();
  await Future<void>.delayed(const Duration(milliseconds: 10));
  if (first.target == null && second.target == null) break;
  if (DateTime.now().isAfter(deadline)) throw StateError('metadata retained capture');
 }
 if ($pcInfo(firstPC).toDartString() != 'fixture.capture|capture.go|7') {
  throw StateError('lost source metadata');
 }
 print('released metadata captures');
}
`)
	run(t, out, "dart", "pub", "get")
	const want = "released metadata captures"
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			got := runLifetimeTarget(t, target, out, "main.dart")
			if got.Exit != 0 || got.Text != want {
				t.Fatalf("%s: %+v", target, got)
			}
		})
	}
}

// Host collection separates retained values from browser collector schedules.
func runLifetimeTarget(t *testing.T, target, out, entry string) programResult {
	t.Helper()
	switch target {
	case "vm":
		return programOutcome(t, out, "dart", entry)
	case "aot":
		exe := filepath.Join(out, "lifetime.exe")
		run(t, out, "dart", "compile", "exe", entry, "-o", exe)
		return programOutcome(t, out, exe)
	case "js":
		run(t, out, "dart", "compile", "js", entry, "-o", "main.js")
		writeFile(t, filepath.Join(out, "collect.js"), "const timer = setInterval(() => global.gc(), 10); timer.unref();\nglobalThis.self = globalThis; require('./main.js');\n")
		return programOutcome(t, out, "node", "--expose-gc", "collect.js")
	case "wasm":
		run(t, out, "dart", "compile", "wasm", entry, "-o", "main.wasm")
		writeFile(t, filepath.Join(out, "collect.mjs"), "const timer = setInterval(() => global.gc(), 10); timer.unref();\n"+wasmRunner)
		return programOutcome(t, out, "node", "--expose-gc", "collect.mjs")
	}
	return programResult{}
}
