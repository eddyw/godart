import importlib.util
import json
from pathlib import Path
import tempfile
import unittest

spec = importlib.util.spec_from_file_location("parity", Path(__file__).with_name("parity.py"))
parity = importlib.util.module_from_spec(spec)
spec.loader.exec_module(parity)


class ParityTest(unittest.TestCase):
    def test_behavior_retains_gaps_duplicates_and_failures(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            manifests = root / "manifests"
            (manifests / "go1").mkdir(parents=True)
            reports = root / "reports"
            reports.mkdir()
            parity.save(manifests / "coverage-manifest.json", [
                dict(package="p", status="vm-suite"), dict(package="q", status="compile-blocked")])
            parity.save(manifests / "go1/os-arch.json", dict(source_sha256="sdk", packages=[
                dict(path=pkg, tests=[dict(name=name, eligible=True, kind="Test") for name in names])
                for pkg, names in [("p", ["TestPass", "TestPartial", "TestMissing"]), ("q", ["TestBlocked"])]]))
            record = dict(package="p", expected=["TestPass", "TestPartial"], go_version="go1", goos="os", goarch="arch",
                          source_sha256="sdk", target="vm", short=True, native=dict(exit=0), gart=dict(exit=0),
                          comparison=dict(passed=["TestPass"], partial=["TestPartial"],
                                          excluded=[dict(test="TestPartial/skip")], problems=[]))
            parity.save(reports / "a.json", record)
            parity.save(reports / "duplicate.json", record)
            groups = parity.behavior([reports], manifests)
            self.assertEqual(len(groups), 1)
            self.assertEqual(len(groups[0]["reports"]), 1)
            self.assertEqual(groups[0]["top_level"], {"passed": 1, "partial": 1})
            self.assertEqual(groups[0]["subtests"], {"excluded": 1})
            self.assertEqual(groups[0]["unobserved_counts"], {"untested": 1, "blocked": 1})
            record["gart"]["exit"] = 1
            parity.save(reports / "failure.json", record)
            failed = parity.behavior([reports], manifests)[0]
            self.assertNotIn("passed", failed["top_level"])
            self.assertTrue(failed["problems"])
            record["short"] = False
            parity.save(reports / "full.json", record)
            self.assertEqual(len(parity.behavior([reports], manifests)), 2)

    def test_application_tests_do_not_count_skips_or_empty_runs_as_passes(self):
        def native(action):
            return json.dumps(dict(Action=action, Test="TestA"))
        self.assertTrue(parity.test_outcomes(native("pass"), "--- PASS: TestA (0.01s)")["complete"])
        self.assertFalse(parity.test_outcomes(native("skip"), "--- SKIP: TestA (0.01s)")["complete"])
        self.assertFalse(parity.test_outcomes("", "")["complete"])
        self.assertFalse(parity.test_outcomes(native("pass"), "")["equal"])
        with self.assertRaises(ValueError):
            parity.test_outcomes(native("pass") + "\n" + native("pass"), "")


if __name__ == "__main__":
    unittest.main()
