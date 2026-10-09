package driver_test

import (
	"path/filepath"
	"testing"
	"time"
)

func compileFixtureTarget(t *testing.T, target, out string) (string, []string) {
	t.Helper()
	var command string
	var commandArgs []string
	runner := "import 'bin/main.dart' as tests;\nvoid main(List<String> args) { tests.main(args); }\n"
	if target == "js" || target == "wasm" {
		runner = `import 'dart:js_interop';
import 'bin/main.dart' as tests;
@JS('godartFixtureArguments')
external JSArray<JSString> get arguments;
void main() {
  tests.main(arguments.toDart.map((argument) => argument.toDart).toList());
}
`
	}
	writeFile(t, filepath.Join(out, "runner.dart"), runner)
	var compiled programResult
	switch target {
	case "aot":
		command = filepath.Join(out, "tests.exe")
		compiled = upstreamOutcome(t, out, nil, runLimit, "dart", "compile", "exe", "runner.dart", "-o", command)
	case "js":
		compiled = upstreamOutcome(t, out, nil, runLimit, "dart", "compile", "js", "runner.dart", "-o", "main.js")
		writeFile(t, filepath.Join(out, "run.js"), "globalThis.self = globalThis;\nglobalThis.godartFixtureArguments = process.argv.slice(2);\nrequire('./main.js');\n")
		command = "node"
		commandArgs = []string{filepath.Join(out, "run.js")}
	case "wasm":
		compiled = upstreamOutcome(t, out, nil, 16*time.Minute, "dart", "compile", "wasm", "runner.dart", "-o", "main.wasm")
		writeFile(t, filepath.Join(out, "run.mjs"), `import {compile} from './main.mjs';
import {readFile} from 'node:fs/promises';
globalThis.godartFixtureArguments = process.argv.slice(2);
const app = await compile(await readFile(new URL('./main.wasm', import.meta.url)));
const instance = await app.instantiate({});
instance.invokeMain();
`)
		command = "node"
		commandArgs = []string{filepath.Join(out, "run.mjs")}
	}
	if compiled.Exit != 0 {
		t.Fatalf("fixture compile %s: %+v", target, compiled)
	}
	return command, commandArgs
}
