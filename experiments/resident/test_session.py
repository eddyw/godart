import json
import os
from pathlib import Path
import re
import shutil
import signal
import subprocess
import tempfile
import time
import unittest
from unittest import mock
import urllib.parse
import urllib.request

from session import CompileError, Session, build, digest


ROOT = Path(__file__).resolve().parents[2]
ENV = dict(os.environ)
ENV.setdefault("GOMAXPROCS", "4")
ENV.setdefault("GOFLAGS", "-p=2")


class ResidentTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.workspace = ROOT / "build/resident-validation"
        cls.workspace.mkdir(parents=True, exist_ok=True)
        cls.gart = cls.workspace / "gart"
        subprocess.run(["go", "build", "-o", str(cls.gart), "./cmd/gart"], cwd=ROOT, env=ENV, check=True)

    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(dir=self.workspace)
        self.root = Path(self.temp.name) / "space $cash #hash"
        self.root.mkdir()
        self.write(".dart_tool/package_config.json", '{"configVersion":2,"packages":[]}')
        self.write("dep.dart", "const value='first';")
        self.write("main.dart", "import 'dep.dart'; void main() { print(value); }")
        self.session = Session(self.root)

    def tearDown(self):
        self.session.close()
        self.temp.cleanup()

    def write(self, name, contents):
        path = self.root / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(contents)
        return path

    def execute(self, want, mode):
        artifact = self.session.compile(self.root / "main.dart")
        result = subprocess.run(["dart", "run", str(artifact)], cwd=self.root,
                                capture_output=True, text=True, timeout=20)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout, want + "\n")
        self.assertEqual(self.session.last["mode"], mode)
        return artifact

    def test_content_invalidation_and_restarted_session(self):
        self.execute("first", "cold")
        self.execute("first", "hit")
        dep = self.root / "dep.dart"
        before = dep.stat()
        dep.write_text("const value='other';")
        os.utime(dep, ns=(before.st_atime_ns, before.st_mtime_ns))
        self.execute("other", "incremental")
        self.assertIn(str(dep), self.session.last["invalidated"])
        self.session.close()
        self.session = Session(self.root)
        self.execute("other", "hit")
        self.assertIsNone(self.session.process)
        self.write("dep.dart", "const value='third';")
        self.execute("third", "cold")

    def test_transitive_import_addition_and_removal(self):
        self.execute("first", "cold")
        self.write("next.dart", "const next='third';")
        self.write("dep.dart", "import 'next.dart'; const value=next;")
        self.execute("third", "incremental")
        self.write("next.dart", "const next='fourth';")
        self.execute("fourth", "incremental")
        (self.root / "next.dart").unlink()
        with self.assertRaises(CompileError):
            self.session.compile(self.root / "main.dart")
        self.assertIsNone(self.session.process)
        self.write("dep.dart", "const value='fixed';")
        self.execute("fixed", "cold")

    def test_runtime_package_and_package_configuration(self):
        first = self.write("runtime-a/lib/runtime.dart", "const value='runtime-a';")
        self.write("runtime-b/lib/runtime.dart", "const value='runtime-b';")
        self.write("main.dart", "import 'package:runtime/runtime.dart'; void main() { print(value); }")

        def configuration(root):
            self.write(".dart_tool/package_config.json", json.dumps(dict(configVersion=2, packages=[
                dict(name="runtime", rootUri=(self.root / root).as_uri() + "/", packageUri="lib/")
            ])))

        configuration("runtime-a")
        self.execute("runtime-a", "cold")
        stamp = first.stat()
        first.write_text("const value='runtime-z';")
        os.utime(first, ns=(stamp.st_atime_ns, stamp.st_mtime_ns))
        self.execute("runtime-z", "incremental")
        configuration("runtime-b")
        self.execute("runtime-b", "cold")

    def test_artifact_and_receipt_corruption(self):
        artifact = self.execute("first", "cold")
        artifact.write_text("corrupt")
        self.execute("first", "incremental")
        artifact.with_suffix(".json").write_text("{}")
        self.execute("first", "incremental")
        artifact.unlink()
        self.execute("first", "incremental")

    def test_compile_error_preserves_last_valid_receipt(self):
        artifact = self.execute("first", "cold")
        receipt = artifact.with_suffix(".json").read_bytes()
        before = digest(artifact)
        self.write("dep.dart", "const value = ;")
        with self.assertRaises(CompileError):
            self.session.compile(self.root / "main.dart")
        self.assertIsNone(self.session.process)
        self.assertEqual(digest(artifact), before)
        self.assertEqual(artifact.with_suffix(".json").read_bytes(), receipt)
        self.write("dep.dart", "const value='fixed';")
        self.execute("fixed", "cold")

    def test_worker_exit_recovers_on_the_next_edit(self):
        self.execute("first", "cold")
        process = self.session.process
        process.kill()
        process.wait()
        self.execute("first", "hit")
        self.write("dep.dart", "const value='after';")
        self.execute("after", "cold")
        self.assertNotEqual(self.session.process.pid, process.pid)
        active = self.session.process
        self.session.close()
        self.assertIsNotNone(active.poll())

    def test_edit_during_request_prevents_publication(self):
        response = self.session._response

        def mutate_after_compile():
            artifact = response()
            path = self.root / "dep.dart"
            stamp = path.stat()
            path.write_text("const value='after';")
            os.utime(path, ns=(stamp.st_atime_ns, stamp.st_mtime_ns))
            return artifact

        with mock.patch.object(self.session, "_response", side_effect=mutate_after_compile):
            with self.assertRaisesRegex(CompileError, "inputs changed"):
                self.session.compile(self.root / "main.dart")
        self.assertFalse(list(self.session.directory.glob("*.json")))
        self.assertIsNone(self.session.process)
        self.execute("after", "cold")

    def test_sdk_identity_and_compiler_defines(self):
        sdk = self.root / ".sdk"
        sdk.mkdir()
        for directory in ["bin", "lib"]:
            (sdk / directory).symlink_to(self.session.sdk / directory, target_is_directory=True)
        shutil.copyfile(self.session.sdk / "version", sdk / "version")
        (sdk / "revision").write_text("probe-one")
        self.session.close()
        self.session = Session(self.root, sdk=sdk, defines={"probe": "one"})
        self.write("main.dart", "void main() { print(const String.fromEnvironment('probe')); }")
        self.execute("one", "cold")
        self.execute("one", "hit")
        self.session.defines["probe"] = "two"
        self.execute("two", "cold")
        (sdk / "revision").write_text("probe-two")
        self.execute("two", "cold")
        (sdk / "version").write_text("3.13.4")
        with self.assertRaisesRegex(CompileError, "requires Dart 3.13.3"):
            self.session.compile(self.root / "main.dart")

    def test_execution_contract(self):
        text = '''import 'dart:io';
void main(List<String> args) {
 var assertions = false;
 assert(() { assertions = true; return true; }());
 print('product=${const bool.fromEnvironment("dart.vm.product")} asserts=$assertions');
 print('${args.join("|")}:${stdin.readLineSync()}:${Directory.current.path}');
 stderr.writeln('error-channel');
 exit(7);
}
'''
        self.write("main.dart", text)
        artifact = self.session.compile(self.root / "main.dart")
        reference = Path(self.temp.name) / "reference.dart"
        reference.write_text(text)
        for assertions in [False, True]:
            args = ["dart", "run"] + (["--enable-asserts"] if assertions else [])
            results = [subprocess.run([*args, str(entry), "two words", "--literal"], cwd=self.temp.name,
                                      input="input\n", text=True, capture_output=True, timeout=20)
                       for entry in [reference, artifact]]
            for result in results:
                self.assertEqual(result.returncode, 7, result.stderr)
                self.assertIn(f"product=false asserts={str(assertions).lower()}", result.stdout)
                self.assertIn("two words|--literal:input:", result.stdout)
                self.assertEqual(result.stderr, "error-channel\n")
            self.assertEqual(results[0].stdout, results[1].stdout)

    def test_go_panic_paths_and_trimpath(self):
        source = self.root / "go-source"
        source.mkdir()
        (source / "go.mod").write_text("module example.com/residentpaths\n\ngo 1.27\n")
        (source / "main.go").write_text('package main\nfunc main() {\npanic("failure")\n}\n')
        output = self.root / "go-output"
        build(self.gart, source, output, ENV)
        with Session(output) as session:
            for trim in [False, False, True]:
                if trim:
                    subprocess.run([str(self.gart), "build", "-trimpath", "-format=false", "-o", str(output), "."],
                                   cwd=source, env=ENV, check=True)
                artifact = session.compile(output / "bin/main.dart")
                result = subprocess.run(["dart", "run", str(artifact)], cwd=output,
                                        text=True, capture_output=True, timeout=20)
                self.assertEqual(result.returncode, 2, result.stdout + result.stderr)
                path = "example.com/residentpaths/main.go" if trim else str(source / "main.go")
                self.assertIn(path + ":3", result.stdout + result.stderr)

    def test_generated_debugger_and_signals(self):
        source = self.root / "debug-source"
        source.mkdir()
        (source / "go.mod").write_text("module example.com/residentdebug\n\ngo 1.27\n")
        (source / "main.go").write_text('package main\nimport("fmt";"time")\nfunc main(){fmt.Println("ready");time.Sleep(2*time.Second);fmt.Println("done")}\n')
        output = Path(self.temp.name) / "debug-output"
        build(self.gart, source, output, ENV)
        with Session(output) as session:
            session.compile(output / "bin/main.dart")
            go_source = source / "main.go"
            go_source.write_text(go_source.read_text().replace('Println("done")', 'Println("next")'))
            build(self.gart, source, output, ENV)
            artifact = session.compile(output / "bin/main.dart")
            self.assertEqual(session.last["mode"], "incremental")
            part = output / "bin/main/main.part.dart"
            line = [i for i, text in enumerate(part.read_text().splitlines(), 1) if "$fmt.Println" in text][1]
            for entry in [output / "bin/main.dart", artifact]:
                log = Path(self.temp.name) / "debug.log"
                with log.open("w") as stream:
                    process = subprocess.Popen(["dart", "run", "--observe=0", "--no-pause-isolates-on-exit", str(entry)],
                                               cwd=output, stdout=stream, stderr=stream, start_new_session=True)
                    try:
                        deadline = time.monotonic() + 20
                        base = None
                        while time.monotonic() < deadline:
                            text = log.read_text()
                            found = re.search(r"The Dart VM service is listening on (http://\S+/)", text)
                            if found and "ready\n" in text:
                                base = found[1]
                                break
                            self.assertIsNone(process.poll(), text)
                            time.sleep(.02)
                        self.assertIsNotNone(base, log.read_text())

                        def rpc(method, **params):
                            with urllib.request.urlopen(base + method + "?" + urllib.parse.urlencode(params), timeout=5) as response:
                                result = json.load(response)
                            self.assertNotIn("error", result)
                            return result["result"]

                        isolate = rpc("getVM")["isolates"][0]["id"]
                        scripts = rpc("getScripts", isolateId=isolate)["scripts"]
                        uri = next(s["uri"] for s in scripts if s["uri"].endswith("/main/main.part.dart"))
                        point = rpc("addBreakpointWithScriptUri", isolateId=isolate, scriptUri=uri, line=line)
                        self.assertTrue(point["resolved"])
                        deadline = time.monotonic() + 10
                        while time.monotonic() < deadline:
                            state = rpc("getIsolate", isolateId=isolate)
                            if state["pauseEvent"]["kind"] == "PauseBreakpoint":
                                break
                            time.sleep(.02)
                        self.assertEqual(state["pauseEvent"]["kind"], "PauseBreakpoint")
                        rpc("resume", isolateId=isolate)
                        self.assertEqual(process.wait(timeout=10), 0, log.read_text())
                        self.assertIn("next\n", log.read_text())
                    finally:
                        if process.poll() is None:
                            os.killpg(process.pid, signal.SIGKILL)
                            process.wait()
            for sig in [signal.SIGINT, signal.SIGTERM]:
                codes = []
                for entry in [output / "bin/main.dart", artifact]:
                    log = Path(self.temp.name) / "signal.log"
                    with log.open("w") as stream:
                        process = subprocess.Popen(["dart", "run", str(entry)], cwd=output,
                                                   stdout=stream, stderr=stream, start_new_session=True)
                        try:
                            deadline = time.monotonic() + 20
                            while "ready\n" not in log.read_text():
                                self.assertLess(time.monotonic(), deadline)
                                self.assertIsNone(process.poll(), log.read_text())
                                time.sleep(.02)
                            os.killpg(process.pid, sig)
                            codes.append(process.wait(timeout=10))
                        finally:
                            if process.poll() is None:
                                os.killpg(process.pid, signal.SIGKILL)
                                process.wait()
                self.assertEqual(codes, [-sig, -sig])


if __name__ == "__main__":
    unittest.main()
