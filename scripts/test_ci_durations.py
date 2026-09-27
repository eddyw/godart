import importlib.util
import json
from pathlib import Path
import tempfile
import unittest

spec = importlib.util.spec_from_file_location("ci_durations", Path(__file__).with_name("ci-durations.py"))
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class DurationTest(unittest.TestCase):
    def test_driver_costs_include_parallel_children_without_double_count(self):
        log = ("=== PAUSE TestParallel/paused\n"
               "=== PAUSE TestParallel/paused/nested\n"
               "=== PAUSE TestParallel/plain/nested\n"
               "=== PAUSE TestParallel/fixedbugs/case\n"
               "--- PASS: TestParallel (13.00s)\n"
               "    --- PASS: TestParallel/paused (5.00s)\n"
               "        --- PASS: TestParallel/paused/nested (4.00s)\n"
               "    --- PASS: TestParallel/plain (3.00s)\n"
               "        --- PASS: TestParallel/plain/nested (7.00s)\n"
               "    --- SKIP: TestParallel/fixedbugs/case (2.00s)\n")
        self.assertEqual(module.driver_costs(log), {"TestParallel": 24.0})

    def test_driver_weights_preserve_mode_and_exclude_incomplete_tests(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            (root / "run.json").write_text(json.dumps(dict(mode="short", source="snapshot")))
            attempt = root / "driver-2/attempt-001"
            attempt.mkdir(parents=True)
            (attempt / "result.json").write_text(json.dumps(dict(exit=1, mode="short", suite="driver")))
            (attempt / "ci.log").write_text(
                "--- PASS: TestSlow (120.00s)\n"
                "    --- PASS: TestSlow/aot (90.00s)\n"
                "--- PASS: TestInstant (0.00s)\n"
                "--- PASS: Example (0.01s)\n"
                "--- SKIP: TestHostOnly (0.02s)\n"
                "--- FAIL: TestFailure (60.00s)\n"
                "=== RUN   TestActive\n")
            active = root / "driver-3/attempt-001"
            active.mkdir(parents=True)
            (active / "ci.log").write_text("--- PASS: TestUnfinishedShard (12.00s)\n")
            weights, evidence = module.measurements(root)
            self.assertEqual(weights, {"driver/short/TestSlow": 120.0,
                                       "driver/short/TestInstant": 0.0,
                                       "driver/short/Example": 0.01,
                                       "driver/short/TestHostOnly": 0.02})
            self.assertEqual(len(evidence["logs"]), 1)
            self.assertEqual(evidence["logs"][0]["exit"], 1)

    def test_target_packages_exclude_incomplete_packages_and_children(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            (root / "run.json").write_text(json.dumps(dict(mode="full", source="snapshot")))
            attempt = root / "upstream-targets-2/attempt-001"
            attempt.mkdir(parents=True)
            (attempt / "result.json").write_text(json.dumps(dict(exit=1, mode="full")))
            (attempt / "ci.log").write_text(
                "    --- PASS: TestUpstreamTargets/encoding/xml (1298.89s)\n"
                "        --- PASS: TestUpstreamTargets/encoding/xml/js (899.07s)\n"
                "    --- FAIL: TestUpstreamTargets/net/http/fcgi (642.78s)\n"
                "        --- PASS: TestUpstreamTargets/net/http/fcgi/js (197.94s)\n")
            weights, evidence = module.measurements(root)
            self.assertEqual(weights, {"TestUpstreamTargets/encoding/xml": 1298.89})
            self.assertEqual(evidence["logs"][0]["exit"], 1)

    def test_selected_packages_exclude_target_children(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            (root / "run.json").write_text(json.dumps(dict(mode="short", source="snapshot")))
            attempt = root / "upstream-priority-1/attempt-001"
            attempt.mkdir(parents=True)
            (attempt / "result.json").write_text(json.dumps(dict(exit=1, mode="short")))
            (attempt / "ci.log").write_text(
                "    --- PASS: TestSelectedUpstream/crypto/subtle (94.00s)\n"
                "        --- PASS: TestSelectedUpstream/crypto/subtle/js (30.00s)\n"
                "    --- FAIL: TestSelectedUpstream/go/parser (50.00s)\n"
                "=== RUN   TestSelectedUpstream/reflect\n")
            weights, evidence = module.measurements(root)
            self.assertEqual(weights, {"TestSelectedUpstream/short/crypto/subtle": 94.0})
            self.assertEqual(evidence["logs"][0]["exit"], 1)

    def test_partial_run_keeps_only_completed_cases(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            (root / "run.json").write_text(json.dumps(dict(mode="full", source="snapshot")))
            attempt = root / "stdlib-0/attempt-001"
            attempt.mkdir(parents=True)
            (attempt / "result.json").write_text(json.dumps(dict(exit=1, mode="full")))
            (attempt / "ci.log").write_text(
                "    --- PASS: TestTestsOfGoroot/index/suffixarray (612.44s)\n"
                "    --- FAIL: TestTestsOfGoroot/crypto/elliptic (20.00s)\n"
                "        --- PASS: TestTestsOfGoroot/index/suffixarray/TestNew32 (200.00s)\n"
                "=== RUN   TestTestsOfGoroot/compress/flate\n")
            active = root / "stdlib-1/attempt-001"
            active.mkdir(parents=True)
            (active / "ci.log").write_text("    --- PASS: TestTestsOfGoroot/os (5.00s)\n")
            weights, evidence = module.measurements(root)
            self.assertEqual(weights, {"TestTestsOfGoroot/index/suffixarray": 612.44})
            self.assertEqual(evidence["logs"][0]["exit"], 1)
            self.assertEqual(len(evidence["logs"][0]["sha256"]), 64)

    def test_short_mode_and_repeated_measurements(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            (root / "run.json").write_text(json.dumps(dict(mode="short", source="snapshot")))
            (root / "result.json").write_text(json.dumps(dict(exit=0)))
            (root / "ci.log").write_text(
                "    --- PASS: TestTestsOfGoroot/index/suffixarray (12.00s)\n"
                "    --- PASS: TestTestsOfGoroot/index/suffixarray (10.00s)\n")
            weights, _ = module.measurements(root)
            self.assertEqual(weights, {"TestTestsOfGoroot/short/index/suffixarray": 12.0})
            (root / "result.json").write_text(json.dumps(dict(exit=0, mode="full")))
            with self.assertRaisesRegex(ValueError, "mode differs"):
                module.measurements(root)


if __name__ == "__main__":
    unittest.main()
