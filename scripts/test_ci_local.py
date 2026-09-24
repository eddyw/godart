import json
import os
import re
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest


class LocalRunnerTest(unittest.TestCase):
    def test_driver_shards_preserve_tests_and_run_resident_tests_once(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            (root / "scripts").mkdir()
            for name in ["ci.sh", "ci-driver.py"]:
                shutil.copy(Path(__file__).with_name(name), root / "scripts" / name)
            durations = root / "internal/driver/testdata/ci-durations.json"
            durations.parent.mkdir(parents=True)
            durations.write_text(json.dumps({"driver/short/TestSlow": 100,
                                            "driver/full/TestNew": 100}))
            (root / "scripts/test_resident.py").write_text('''import unittest
from pathlib import Path
class ResidentTest(unittest.TestCase):
    def test_run(self):
        with Path("resident.txt").open("a") as output:
            output.write("run\\n")
''')
            binary = root / "bin"
            binary.mkdir()
            go = binary / "go"
            go.write_text('''#!/usr/bin/env python3
import json, sys
if "-list" in sys.argv:
    print("TestSlow\\nTestNew\\nTestPrograms\\nExample\\nFuzzInput\\nok package 0.1s")
else:
    with open("calls.jsonl", "a") as output:
        output.write(json.dumps(sys.argv[1:]) + "\\n")
''')
            go.chmod(0o755)
            env = dict(os.environ, PATH=str(binary) + os.pathsep + os.environ["PATH"])
            env.pop("GITHUB_STEP_SUMMARY", None)
            names = ["TestSlow", "TestNew", "Example", "FuzzInput"]
            for mode in ["short", "full"]:
                env["GART_UPSTREAM_MODE"] = mode
                env["GART_PARITY_REPORT_DIR"] = str(root / mode)
                for index in range(3):
                    subprocess.run(["bash", "scripts/ci.sh", "driver", str(index), "3"],
                                   cwd=root, env=env, check=True, capture_output=True)
                calls = [json.loads(line) for line in (root / "calls.jsonl").read_text().splitlines()][-3:]
                selected = []
                for index, args in enumerate(calls):
                    self.assertIn("-short=" + str(mode == "short").lower(), args)
                    pattern = args[args.index("-run") + 1]
                    self.assertIsNone(re.fullmatch(pattern, "TestPrograms"))
                    self.assertIsNone(re.fullmatch(pattern, "TestSlowExtra"))
                    selected.extend(name for name in names if re.fullmatch(pattern, name))
                    record = json.loads((root / mode / f"driver-selection-{index}.json").read_text())
                    self.assertCountEqual(record["available"], names)
                    self.assertCountEqual(record["selected"], [name for name in names if re.fullmatch(pattern, name)])
                self.assertCountEqual(selected, names)
            self.assertEqual((root / "resident.txt").read_text().splitlines(), ["run", "run"])

    def test_priority_shards_run_shared_helpers_once(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            (root / "scripts").mkdir()
            shutil.copy(Path(__file__).with_name("ci.sh"), root / "scripts/ci.sh")
            binary = root / "bin"
            binary.mkdir()
            go = binary / "go"
            go.write_text('''#!/usr/bin/env python3
import json, sys
with open("calls.jsonl", "a") as output:
    output.write(json.dumps(sys.argv[1:]) + "\\n")
''')
            go.chmod(0o755)
            env = dict(os.environ, PATH=str(binary) + os.pathsep + os.environ["PATH"],
                       GART_UPSTREAM_MODE="short")
            env.pop("GITHUB_STEP_SUMMARY", None)
            for index in range(4):
                subprocess.run(["bash", "scripts/ci.sh", "upstream-priority", str(index), "4"],
                               cwd=root, env=env, check=True, capture_output=True)
            calls = [json.loads(line) for line in (root / "calls.jsonl").read_text().splitlines()]
            policy = [args for args in calls if "./internal/parity" in args]
            driver = [args for args in calls if "./internal/driver" in args]
            self.assertEqual(len(policy), 1)
            self.assertEqual(len(driver), 4)
            for index, args in enumerate(driver):
                self.assertIn("-short=true", args)
                self.assertEqual(args[args.index("-shard-index") + 1], str(index))
                self.assertEqual(args[args.index("-shard-count") + 1], "4")
                selection = args[args.index("-run") + 1]
                self.assertIn("TestSelectedUpstream", selection)
                for helper in ("TestHTTPUpstream", "TestHTTPNativeServer", "TestParityHarness"):
                    self.assertEqual(helper in selection, index == 0)

    def test_retry_preserves_attempts_and_rejects_source_drift(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            (root / "scripts").mkdir()
            shutil.copy(Path(__file__).with_name("ci-local.py"), root / "scripts/ci-local.py")
            (root / ".gitignore").write_text("/build/\n")
            (root / "Makefile").write_text("get:\n\t@true\n")
            (root / "scripts/ci-matrix.json").write_text(json.dumps({"include": [
                dict(suite=suite, shard=0, shards=1, timeout=1, job_timeout=1)
                for suite in ["pass", "retry"]
            ]}))
            (root / "scripts/ci.sh").write_text('''set -eu
mkdir -p build
printf 'evidence\\n' > "$GART_PARITY_REPORT_DIR/result.txt"
if [ "$1" = retry ] && [ ! -e build/attempted ]; then
  touch build/attempted
  exit 23
fi
''')
            subprocess.run(["git", "init", "-q", str(root)], check=True)
            subprocess.run(["git", "add", "."], cwd=root, check=True)
            command = [sys.executable, "scripts/ci-local.py", "--output", "build/results"]

            def run(*extra):
                return subprocess.run(command + list(extra), cwd=root, text=True, capture_output=True)

            first = run()
            self.assertEqual(first.returncode, 1, first.stdout + first.stderr)
            self.assertTrue((root / "build/results/retry-0/attempt-001/reports/result.txt").is_file())
            resumed = run("--resume")
            self.assertEqual(resumed.returncode, 0, resumed.stdout + resumed.stderr)
            self.assertIn("REUSE pass-0", resumed.stdout)
            results = json.loads((root / "build/results/results.json").read_text())
            self.assertEqual([r["exit"] for r in results], [0, 0])
            self.assertEqual(results[1]["attempt"], "retry-0/attempt-002")
            previous = json.loads((root / "build/results/retry-0/attempt-001/result.json").read_text())
            self.assertEqual(previous["exit"], 23)
            (root / "Makefile").write_text("get:\n\t@echo changed\n")
            drift = run("--resume")
            self.assertEqual(drift.returncode, 2)
            self.assertIn("source, environment, or selection changed", drift.stderr)


if __name__ == "__main__":
    unittest.main()
