import importlib.util
import json
from pathlib import Path
import tempfile
import unittest

spec = importlib.util.spec_from_file_location("parity", Path(__file__).with_name("parity.py"))
parity = importlib.util.module_from_spec(spec)
spec.loader.exec_module(parity)


class ParityTest(unittest.TestCase):
    def test_fuzz_replay_requires_success_and_retained_inputs(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            fixture = root / "fixture" / "input"
            fixture.parent.mkdir()
            fixture.write_text("upstream assertion and input")
            outcomes = {"FuzzExample": "pass", "FuzzExample/input": "pass"}
            process = dict(exit=0, outcomes=outcomes)
            report = dict(kind="fuzz-replay", status="passed", generation=dict(exit=0),
                          package="p", function="FuzzExample", go_version="go1", dart_version="dart1",
                          goos="linux", goarch="amd64", source_sha256="source", fixture_sha256="fixture",
                          native=process, gart=dict(process), comparison=dict(passed=list(outcomes)),
                          fixtures=[dict(path="input", sha256=parity.digest(fixture))])
            path = root / "fuzz.json"
            parity.save(path, report)
            self.assertEqual(parity.fuzz_behavior([root])[0]["status"], "passed")
            report["gart"] = dict(exit=0, outcomes={"FuzzExample": "pass"})
            parity.save(path, report)
            self.assertEqual(parity.fuzz_behavior([root])[0]["status"], "failed")
            report["gart"] = process
            report["generation"]["exit"] = 1
            parity.save(path, report)
            self.assertEqual(parity.fuzz_behavior([root])[0]["status"], "failed")
            report["generation"]["exit"] = 0
            parity.save(path, report)
            fixture.write_text("changed")
            self.assertEqual(parity.fuzz_behavior([root])[0]["status"], "failed")
            report["status"] = "incomplete"
            parity.save(path, report)
            self.assertEqual(parity.fuzz_behavior([root])[0]["status"], "incomplete")

    def test_application_shards_preserve_the_complete_corpus(self):
        cases = [dict(name=name) for name in ["c", "a", "b", "d", "e"]]
        for count in [1, 3, 8]:
            shards = [parity.application_shard(cases, index, count) for index in range(count)]
            actual = [case["name"] for shard in shards for case in shard]
            self.assertEqual(sorted(actual), ["a", "b", "c", "d", "e"])
            for index, shard in enumerate(shards):
                self.assertEqual(shard, parity.application_shard(list(reversed(cases)), index, count))
        for index, count in [(-1, 1), (1, 1), (0, 0)]:
            with self.assertRaises(ValueError):
                parity.application_shard(cases, index, count)

    def test_language_execution_requires_original_output_and_success(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            fixture = dict(name="p", file="p.go")
            execution = dict(ran=True, exit=0, timed_out=False, output_sha256="original")
            outcome = dict(fixture=fixture, status="passed", native=execution, gart=dict(execution), expected_sha256="original")
            report = dict(kind="language", go_version="go1", complete=True, outcomes=[outcome], shard=0, shards=1)
            path = root / "language.json"
            parity.save(path, report)
            self.assertEqual(parity.language_behavior([root])[0]["counts"], {"passed": 1})
            outcome["gart"]["output_sha256"] = "different"
            parity.save(path, report)
            result = parity.language_behavior([root])[0]
            self.assertEqual(result["counts"], {"failed": 1})
            self.assertTrue(result["problems"])
            report["complete"] = False
            parity.save(path, report)
            self.assertEqual(parity.language_behavior([root])[0]["counts"], {"unobserved": 1})

    def test_language_inventory_never_implies_an_execution_pass(self):
        cases = [dict(name="run", action="run", native_eligible=True, gart_eligible=True, adapter="single-file"),
                 dict(name="compile", action="compile", native_eligible=True, gart_eligible=True, adapter="missing"),
                 dict(name="blocked", action="run", native_eligible=True, gart_eligible=True,
                      adapter="single-file", rejection=["unsupported"]),
                 dict(name="deep", action="run", native_eligible=True, gart_eligible=True,
                      adapter="single-file", exclusion=dict(classification="implementation-gap", reason="deep recursion")),
                 dict(name="platform", action="run", native_eligible=False, gart_eligible=True, adapter="single-file")]
        result = parity.language_assessment(dict(cases=cases))
        self.assertEqual(result["counts"], {"unmeasured": 1, "adapter-missing": 1, "known-rejection": 1,
                                           "implementation-gap": 1, "ineligible": 1})
        self.assertEqual(result["actions"], {"run": 4, "compile": 1})

    def test_compile_recipe_requires_both_compilers_and_wrapper_identity(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            execution = dict(ran=True, exit=0, timed_out=False, output_sha256="diagnostics")
            outcome = dict(fixture=dict(name="p", recipe="compile"), contract="compile", status="passed",
                           wrapper_sha256="wrapper", native=dict(execution), gart=dict(execution))
            report = dict(kind="language", complete=True, outcomes=[outcome], shard=0, shards=1)
            path = root / "language.json"
            parity.save(path, report)
            self.assertEqual(parity.language_behavior([root])[0]["counts"], {"passed": 1})
            self.assertEqual(parity.language_behavior([root])[0]["configuration"]["contract"], "compile")
            for field, value in [("ran", False), ("exit", 1), ("timed_out", True)]:
                outcome["gart"] = {**execution, field: value}
                parity.save(path, report)
                self.assertEqual(parity.language_behavior([root])[0]["counts"], {"failed": 1})
            outcome["gart"] = dict(execution)
            outcome["wrapper_sha256"] = ""
            parity.save(path, report)
            self.assertEqual(parity.language_behavior([root])[0]["counts"], {"failed": 1})
            outcome["wrapper_sha256"] = "wrapper"
            report["contract"] = "run"
            parity.save(path, report)
            self.assertEqual(parity.language_behavior([root])[0]["counts"], {"failed": 1})
            report["contract"] = "compile"
            outcome["fixture"]["recipe"] = "run"
            parity.save(path, report)
            self.assertEqual(parity.language_behavior([root])[0]["counts"], {"failed": 1})


    def test_scope_review_does_not_hide_new_packages_or_sdk_changes(self):
        with tempfile.TemporaryDirectory() as temporary:
            path = Path(temporary) / "scope.json"
            parity.save(path, dict(go_version="go1", basis="Source review only", packages=[
                dict(package="p", assessment="mixed", reason="Portable APIs and native limits")]))
            api = dict(go_version="go1", unassessed_packages=["p", "new"])
            result = parity.scope_assessment(api, path)
            self.assertEqual(result["counts"], {"current": 1, "unassessed": 1})
            self.assertEqual(result["packages"][0]["assessment"], "mixed")
            api["go_version"] = "go2"
            self.assertEqual(parity.scope_assessment(api, path)["counts"], {"needs-review": 1, "unassessed": 1})

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
                                          excluded=[dict(test="TestPartial/skip", reason="no native allocation counter")], problems=[]))
            parity.save(reports / "a.json", record)
            parity.save(reports / "duplicate.json", record)
            groups = parity.behavior([reports], manifests)
            self.assertEqual(len(groups), 1)
            self.assertEqual(len(groups[0]["reports"]), 1)
            self.assertEqual(groups[0]["top_level"], {"passed": 1, "partial": 1})
            self.assertEqual(groups[0]["subtests"], {"excluded": 1})
            excluded = next(x for x in groups[0]["outcomes"] if x["name"] == "TestPartial/skip")
            self.assertEqual(excluded["exclusion_reasons"], ["no native allocation counter"])
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
