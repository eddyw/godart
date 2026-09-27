import copy
import importlib.util
import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest import mock

spec = importlib.util.spec_from_file_location("parity", Path(__file__).with_name("parity.py"))
parity = importlib.util.module_from_spec(spec)
spec.loader.exec_module(parity)


class ParityTest(unittest.TestCase):
    def test_supplied_binary_has_no_assumed_source_revision(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            binary = root / "gart"
            binary.write_bytes(b"compiler fixture")
            args = parity.argparse.Namespace(
                output=root / "report", project=root, gart=binary,
                mode="compile", pattern=".", tags="", timeout=1)
            with mock.patch.object(parity, "identity", return_value={"head": "unrelated"}), \
                 mock.patch.object(parity, "command", return_value={"exit": 0}), \
                 mock.patch.object(parity.subprocess, "check_output", return_value="version"), \
                 mock.patch.object(parity.platform, "platform", return_value="fixture"), \
                 mock.patch("builtins.print"):
                self.assertEqual(parity.application(args), 0)
            result = parity.read(args.output / "application.json")
            self.assertIsNone(result["compiler"])
            self.assertEqual(result["gart_sha256"], parity.digest(binary))

    def test_application_suite_uses_the_build_source_identity(self):
        for changed in (False, True):
            with self.subTest(changed=changed), tempfile.TemporaryDirectory() as temporary:
                args = parity.argparse.Namespace(
                    output=Path(temporary) / "report", shard_index=0, shard_count=1, timeout=1)
                before = {"head": "base", "source_sha256": "source"}
                after = {"head": "base", "source_sha256": "different" if changed else "source"}
                cases = [{"name": "fixture", "project": "fixture", "pattern": ".", "mode": "test", "tags": ""}]
                with mock.patch.object(parity, "read", return_value=cases), \
                     mock.patch.object(parity, "identity", side_effect=[before, after]), \
                     mock.patch.object(parity, "command", return_value={"exit": 0}), \
                     mock.patch.object(parity, "application", return_value=0) as application:
                    self.assertEqual(parity.applications(args), 0)
                    self.assertEqual(application.call_args.args[1], None if changed else before)

    def test_identity_does_not_attribute_ignored_source(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            subprocess.run(["git", "init", "-q", root], check=True)
            (root / ".gitignore").write_text("ignored/\n")
            ignored = root / "ignored"
            ignored.mkdir()
            (ignored / "main.go").write_text("package main\nfunc main() {}\n")
            result = parity.identity(ignored)
            self.assertIsNone(result["head"])
            self.assertIsNone(result["source_sha256"])

    def test_identity_preserves_nested_module_source(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            subprocess.run(["git", "init", "-q", root], check=True)
            nested = root / "module"
            nested.mkdir()
            source = nested / "main.go"
            source.write_text("package main\nfunc main() {}\n")
            before = parity.identity(nested)["source_sha256"]
            self.assertIsNotNone(before)
            source.write_text('package main\nfunc main() { println("changed") }\n')
            self.assertNotEqual(before, parity.identity(nested)["source_sha256"])

    def test_generated_program_requires_original_recipe_and_source(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source = "package main\r\nfunc main() {}\r\n"
            generator = dict(ran=True, exit=0, timed_out=False,
                             output_sha256=parity.hashlib.sha256(source.encode()).hexdigest())
            execution = dict(ran=True, exit=0, timed_out=False, output_sha256="expected")
            case = dict(fixture=dict(name="p", recipe="runoutput"), contract="runoutput", status="passed",
                        generator=generator, generated_source=source, expected_sha256="expected",
                        native=dict(execution), gart=dict(execution))
            report = dict(kind="language", contract="runoutput", complete=True, outcomes=[case], shard=0, shards=1)
            path = root / "language.json"
            parity.save(path, report)
            self.assertEqual(parity.language_behavior([root])[0]["counts"], {"passed": 1})
            mutations = [
                lambda c: c.pop("generator"),
                lambda c: c["generator"].update(ran=False),
                lambda c: c["generator"].update(exit=1),
                lambda c: c["generator"].update(timed_out=True),
                lambda c: c.update(generated_source=source.replace("\r\n", "\n")),
                lambda c: c["fixture"].update(recipe="runoutput extra.go"),
                lambda c: c.update(build_error="invalid source"),
                lambda c: c["gart"].update(output_sha256="different"),
            ]
            for mutate in mutations:
                changed = copy.deepcopy(report)
                mutate(changed["outcomes"][0])
                parity.save(path, changed)
                self.assertEqual(parity.language_behavior([root])[0]["counts"], {"failed": 1})

    def test_generator_companions_require_source_and_command_evidence(self):
        fixture = dict(file="rotate0.go", recipe="runoutput ./rotate.go", generator_files=["rotate.go"],
                       files=[dict(path=name, sha256="a" * 64) for name in ["rotate0.go", "rotate.go"]])
        case = dict(fixture=fixture, generator=dict(command=["go", "run", "-gcflags=all=",
                    "/sdk/test/rotate0.go", "/sdk/test/rotate.go"]))
        self.assertTrue(parity.generator_recipe_valid(case))
        for mutate in [
            lambda c: c["fixture"].pop("generator_files"),
            lambda c: c["fixture"].update(recipe="runoutput ../rotate.go"),
            lambda c: c["fixture"].update(recipe="runoutput -gcflags=-N"),
            lambda c: c["fixture"]["files"].pop(),
            lambda c: c["fixture"]["files"][1].update(sha256=""),
            lambda c: c["generator"]["command"].pop(),
            lambda c: c["generator"]["command"].__setitem__(4, "/other/rotate.go"),
            lambda c: c["generator"]["command"].__setitem__(0, "echo"),
        ]:
            changed = copy.deepcopy(case)
            mutate(changed)
            self.assertFalse(parity.generator_recipe_valid(changed))

    def test_importer_inputs_preserve_assertions_and_sdk_identity(self):
        import hashlib
        digest = lambda text: hashlib.sha256(text.encode()).hexdigest()
        template = r'{{with context}}{{printf "{\"compiler\":%q,\"goos\":%q,\"goarch\":%q,\"goroot\":%q,\"cgo_enabled\":%t" .Compiler .GOOS .GOARCH .GOROOT .CgoEnabled}},"build_tags":[{{range $i, $tag := .BuildTags}}{{if $i}},{{end}}{{printf "%q" $tag}}{{end}}],"tool_tags":[{{range $i, $tag := .ToolTags}}{{if $i}},{{end}}{{printf "%q" $tag}}{{end}}],"release_tags":[{{range $i, $tag := .ReleaseTags}}{{if $i}},{{end}}{{printf "%q" $tag}}{{end}}]}{{end}}'
        guard = '\tif runtime.Compiler != "gc" {\n\t\tt.Skipf("gc-built packages not available (compiler = %s)", runtime.Compiler)\n\t}'
        initializer = "var importer = New(&build.Default, token.NewFileSet(), make(map[string]*types.Package))"
        assertion = "if answer != 42 { t.Fatal(answer) }"
        context = dict(compiler="gc", goos="darwin", goarch="arm64", goroot="/sdk", cgo_enabled=True,
                       build_tags=[], tool_tags=["arm64.v8.0"], release_tags=["go1." + str(n) for n in range(1, 28)])
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            for pkg in ["go/internal/gcimporter", "go/internal/srcimporter"]:
                if pkg.endswith("/gcimporter"):
                    source = '\n\t"runtime"\n' + guard + "\n" + assertion
                    adapted = source.replace(guard, guard.replace("runtime.Compiler", '"gc"')).replace('\n\t"runtime"\n', "\n")
                else:
                    source = 'func TestCgo(t *testing.T) { testenv.MustHaveCGO(t) }\n' + initializer + "\n" + assertion
                    setup = ('var _ = func() bool {\n\tbuild.Default.Compiler = "gc"\n\tbuild.Default.GOARCH = "arm64"\n'
                             '\tbuild.Default.CgoEnabled = true\n\tbuild.Default.BuildTags = []string{}\n'
                             '\tbuild.Default.ToolTags = []string{"arm64.v8.0"}\n\tbuild.Default.ReleaseTags = []string{' +
                             ", ".join(json.dumps(v) for v in context["release_tags"]) + '}\n\treturn true\n}()\n\n')
                    adapted = source.replace(initializer, setup + initializer).replace(
                        "testenv.MustHaveCGO(t)", 'if !build.Default.CgoEnabled { t.Skip("selected Go SDK has no cgo") }')
                name = pkg.rsplit("/", 1)[1] + "_test.go"
                fixture = dict(command=["go", "list", "-f=" + template, "runtime"], output=json.dumps(context),
                               context=context, source=dict(path="src/" + pkg + "/" + name, source=source,
                               sha256=digest(source), adapted=adapted, adapted_sha256=digest(adapted)))
                report = dict(package=pkg, go_version="go1.27.1", goos="darwin", goarch="arm64",
                              environment=dict(GOROOT="/sdk"), importer_inputs=fixture, cgo=True, build_tags=[], tool_tags=["arm64.v8.0"])
                inventory = dict(packages=[dict(path=pkg, files=[dict(path=name, sha256=digest(source))])])
                parity.save(root / "go1.27.1/darwin-arm64.json", inventory)
                self.assertIsNone(parity.importer_input_problem(report, root))
                for mutate in [
                    lambda r: r.pop("importer_inputs"),
                    lambda r: r["importer_inputs"].update(command=["echo", "gc"]),
                    lambda r: r["importer_inputs"].update(output="invalid json"),
                    lambda r: r["importer_inputs"].update(context=dict(context, compiler="gart")),
                    lambda r: r["importer_inputs"].update(context=dict(context, cgo_enabled=False)),
                    lambda r: r["importer_inputs"].update(context=dict(context, tool_tags=[])),
                    lambda r: r["environment"].update(GOROOT="/other"),
                    lambda r: r["importer_inputs"]["source"].update(path="src/other.go"),
                    lambda r: r["importer_inputs"]["source"].update(source="changed assertions"),
                    lambda r: r["importer_inputs"]["source"].update(adapted="changed assertions"),
                    lambda r: r["importer_inputs"]["source"].update(adapted=adapted.replace("42", "41"),
                                                                          adapted_sha256=digest(adapted.replace("42", "41"))),
                    lambda r: r["importer_inputs"]["source"].update(source=source.replace("42", "41"),
                                                                          sha256=digest(source.replace("42", "41"))),
                ]:
                    changed = copy.deepcopy(report)
                    mutate(changed)
                    self.assertIsNotNone(parity.importer_input_problem(changed, root))

    def test_resolver_inputs_preserve_native_setup(self):
        import copy
        import hashlib
        digest = lambda text: hashlib.sha256(text.encode()).hexdigest()
        source = "package net\nfunc setupTestData() {}\n"
        generator = 'package net\nimport gartjson "encoding/json"\n\nfunc setupTestData() {}\nprobe'
        inputs = dict(TCP=[], UDP=[], IP=[])
        output = "GART_RESOLVER_INPUTS " + json.dumps(inputs) + "\n"
        fixture = dict(source=source, source_sha256=digest(source), generator_source=generator,
                       generator_sha256=digest(generator), fixture_source="fixture", fixture_sha256=digest("fixture"),
                       output=output, inputs=inputs,
                       generator=dict(ran=True, exit=0, timed_out=False, output_sha256=digest(output),
                                      command=["go", "test", "-overlay=inputs.json", "-run=^TestGartResolverInputs$"]))
        report = dict(resolver_inputs=fixture, comparison=dict(passed=["TestResolveTCPAddr"]))
        self.assertIsNone(parity.resolver_input_problem(report, "net-selected"))
        for mutate in [
            lambda r: r.pop("resolver_inputs"),
            lambda r: r["resolver_inputs"]["generator"].update(exit=1),
            lambda r: r["resolver_inputs"]["generator"].update(ran=False),
            lambda r: r["resolver_inputs"]["generator"].update(command=["echo", "fake"]),
            lambda r: r["resolver_inputs"].update(source="changed"),
            lambda r: r["resolver_inputs"].update(fixture_source="changed"),
            lambda r: r["resolver_inputs"].update(output="changed"),
            lambda r: r["resolver_inputs"]["inputs"].update(IP=[dict(Address="changed")]),
        ]:
            changed = copy.deepcopy(report)
            mutate(changed)
            self.assertIsNotNone(parity.resolver_input_problem(changed, "net-selected"))
        changed = copy.deepcopy(report)
        changed["resolver_inputs"]["generator_source"] = "different setup"
        changed["resolver_inputs"]["generator_sha256"] = digest("different setup")
        self.assertIsNotNone(parity.resolver_input_problem(changed, "net-selected"))
        partial = dict(comparison=dict(passed=[], partial=["TestResolveTCPAddr"]))
        self.assertIsNone(parity.resolver_input_problem(partial, "net-selected"))

    def test_certificate_inputs_preserve_setup_and_require_success(self):
        expressions = [f"generatePEMCertWithRepeatSAN(100, {i}, \"fake.dns\")" for i in range(4)]
        source = "\n".join(f"var x{i} = {expression}" for i, expression in enumerate(expressions))
        pem = "-----BEGIN CERTIFICATE-----\nAA==\n-----END CERTIFICATE-----\n"
        digest = lambda text: parity.hashlib.sha256(text.encode()).hexdigest()
        inputs = [dict(start=source.index(expression), end=source.index(expression) + len(expression),
                       expression=expression, pem=pem) for expression in expressions]
        adapted = source
        for item in reversed(inputs):
            adapted = adapted[:item["start"]] + json.dumps(pem) + adapted[item["end"]:]
        output = json.dumps([pem] * 4)
        fixture = dict(source=source, source_sha256=digest(source), generator_source="original helper",
                       generator_sha256=digest("original helper"), output=output, inputs=inputs,
                       adapted_sha256=digest(adapted),
                       generator=dict(ran=True, exit=0, timed_out=False, output_sha256=digest(output)))
        report = dict(native_args=["test", "-overlay=fixture.json"], certificate_inputs=fixture)
        self.assertIsNone(parity.certificate_input_problem(report))
        mutations = [
            lambda r: r.update(native_args=["test"]),
            lambda r: r.pop("certificate_inputs"),
            lambda r: r["certificate_inputs"]["generator"].update(exit=1),
            lambda r: r["certificate_inputs"]["generator"].update(ran=False),
            lambda r: r["certificate_inputs"]["generator"].update(timed_out=True),
            lambda r: r["certificate_inputs"].update(source="changed source"),
            lambda r: r["certificate_inputs"].update(generator_source="changed helper"),
            lambda r: r["certificate_inputs"].update(adapted_sha256="changed assertions"),
            lambda r: r["certificate_inputs"]["inputs"][0].update(pem="another certificate"),
            lambda r: r["certificate_inputs"]["inputs"][0].update(start=1),
        ]
        for mutate in mutations:
            changed = copy.deepcopy(report)
            mutate(changed)
            self.assertIsNotNone(parity.certificate_input_problem(changed))
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            manifests = root / "manifests"
            manifests.mkdir()
            parity.save(manifests / "coverage-manifest.json", [])
            report.update(package="crypto/x509", native=dict(exit=0), gart=dict(exit=0),
                          expected=["TestGoVerify"], comparison=dict(passed=["TestGoVerify"]))
            parity.save(root / "report.json", report)
            self.assertEqual(parity.behavior([root], manifests)[0]["top_level"], {"passed": 1})
            report["certificate_inputs"]["generator"]["exit"] = 1
            parity.save(root / "report.json", report)
            self.assertNotIn("passed", parity.behavior([root], manifests)[0]["top_level"])

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

    def test_directory_diagnostics_require_ordered_compilation_and_matchers(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            module = "module test\n\ngo 1.27.1\n"
            diagnostic = "b.go:3: undefined: Missing"
            execution = dict(ran=True, exit=0, timed_out=False,
                             output_sha256=parity.hashlib.sha256(b"").hexdigest())
            first = dict(execution, command=["go", "tool", "compile", "-e", "-d=panic", "sdk/p.dir/a.go"])
            last = dict(execution, exit=2, output_sha256=parity.hashlib.sha256(diagnostic.encode()).hexdigest(),
                        command=["go", "tool", "compile", "-e", "-d=panic", "sdk/p.dir/b.go"])
            case = dict(fixture=dict(name="p", recipe="errorcheckdir", adapter="directory-packages",
                                     files=[dict(path="p.dir/a.go", sha256="a"), dict(path="p.dir/b.go", sha256="b")]),
                        contract="errorcheckdir", status="passed", build_error=diagnostic,
                        native=dict(last), gart=dict(last), native_match=dict(execution), gart_match=dict(execution),
                        native_diagnostics=diagnostic, gart_diagnostics=diagnostic, wrapper_sha256="original matcher",
                        module_source=module, module_sha256=parity.hashlib.sha256(module.encode()).hexdigest(),
                        directory_packages=[dict(name="a", path="test/a", files=["a.go"]),
                                            dict(name="b", path="test/b", files=["b.go"])],
                        adapted_sources={"a/a.go": "package a\n", "b/b.go": "package b\n"},
                        native_compile=[first, last], importcfg_sha256="config")
            path = root / "language.json"

            def assess(value, status):
                parity.save(path, dict(kind="language", go_version="go1.27.1", contract="errorcheckdir",
                                       complete=True, outcomes=[value], shard=0, shards=1))
                self.assertEqual(parity.language_behavior([root])[0]["counts"], {status: 1})

            assess(case, "passed")
            for mutate in [
                lambda c: c["native_compile"][0].update(exit=1),
                lambda c: c["native_compile"][1].update(exit=0),
                lambda c: c["native_compile"][0]["command"].remove("-d=panic"),
                lambda c: c["native_compile"][0].update(timed_out=True),
                lambda c: c["native"].update(exit=1),
                lambda c: c["gart"].update(exit=0),
                lambda c: c["gart_match"].update(exit=1),
                lambda c: c.update(adapted_sources={}),
                lambda c: c.update(module_sha256="changed"),
                lambda c: c["fixture"].update(recipe="errorcheckdir -0"),
                lambda c: c.update(gart_diagnostics="changed"),
            ]:
                changed = copy.deepcopy(case)
                mutate(changed)
                assess(changed, "failed")

    def test_combined_directory_requires_rejection_recompile_and_runtime(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            module = "module test\n\ngo 1.27.1\n"
            diagnostic = "b.go:3: undefined: Missing"
            digest = lambda text: parity.hashlib.sha256(text.encode()).hexdigest()
            execution = dict(ran=True, exit=0, timed_out=False, output_sha256=digest(""))
            compiled = [dict(execution, command=["go", "tool", "compile", "-e", "-d=panic", "sdk/p.dir/" + f + ".go"])
                        for f in ("a", "b", "main")]
            compiled[1].update(exit=2, output_sha256=digest(diagnostic))
            case = dict(fixture=dict(name="p", recipe="errorcheckandrundir -1", adapter="directory-packages",
                                     files=[dict(path="p.dir/" + f + ".go") for f in ("a", "b", "main")]),
                        contract="errorcheckandrundir", status="passed", build_error=diagnostic,
                        native=dict(execution), gart=dict(execution), expected_sha256=digest(""),
                        gart_reject=dict(compiled[1]), native_match=dict(execution), gart_match=dict(execution),
                        native_diagnostics=diagnostic, gart_diagnostics=diagnostic, wrapper_sha256=digest("original matcher"),
                        module_source=module, module_sha256=digest(module), importcfg_sha256="config",
                        directory_packages=[dict(name=f, path="test/" + f, files=[f + ".go"]) for f in ("a", "b", "main")],
                        adapted_sources={f + "/" + f + ".go": "package " + f + "\n" for f in ("a", "b", "main")},
                        native_compile=compiled, native_recompile=copy.deepcopy(compiled),
                        native_link=dict(execution, command=["go", "tool", "link", "main.o"]))
            path = root / "language.json"

            def assess(value, status):
                parity.save(path, dict(kind="language", go_version="go1.27.1", contract="errorcheckandrundir",
                                       complete=True, outcomes=[value], shard=0, shards=1))
                self.assertEqual(parity.language_behavior([root])[0]["counts"], {status: 1})

            assess(case, "passed")
            for mutate in [
                lambda c: c["fixture"].update(recipe="errorcheckandrundir -1 -m"),
                lambda c: c["fixture"].update(recipe="errorcheckandrundir"),
                lambda c: c["native_compile"][0].update(exit=2),
                lambda c: c["native_compile"][1].update(exit=0),
                lambda c: c["native_compile"][2].update(exit=2),
                lambda c: c["native_compile"][0]["command"].remove("-d=panic"),
                lambda c: c.update(native_recompile=[]),
                lambda c: c["native_recompile"][0].update(command=["go", "build"]),
                lambda c: c["native_recompile"][1].update(exit=0),
                lambda c: c["native_recompile"][2].update(timed_out=True),
                lambda c: c.update(gart_reject=None),
                lambda c: c["gart_reject"].update(exit=0),
                lambda c: c["gart_match"].update(exit=1),
                lambda c: c.update(native_match=None),
                lambda c: c.update(wrapper_sha256=""),
                lambda c: c.update(build_error=""),
                lambda c: c.update(gart_diagnostics="changed"),
                lambda c: c["native"].update(ran=False),
                lambda c: c["gart"].update(exit=1),
                lambda c: c["gart"].update(output_sha256="changed"),
                lambda c: c["native_link"].update(exit=1),
                lambda c: c.update(adapted_sources={}),
            ]:
                changed = copy.deepcopy(case)
                mutate(changed)
                assess(changed, "failed")

    def test_package_directories_require_compile_and_source_evidence(self):
        import hashlib
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            module = "module test\n\ngo 1.27.1\n"
            execution = dict(ran=True, exit=0, timed_out=False, output_sha256="expected")
            fixture = dict(name="p", recipe="compiledir", adapter="directory-packages",
                           files=[dict(path="p.dir/a.go", sha256="source")])
            compiled = dict(execution, command=["go", "tool", "compile", "-e", "sdk/p.dir/a.go"])
            case = dict(fixture=fixture, contract="compiledir", status="passed",
                        native=dict(execution), gart=dict(execution),
                        wrapper_source='import "bin/main.dart";\nvoid main() {}\n',
                        wrapper_sha256=hashlib.sha256(b'import "bin/main.dart";\nvoid main() {}\n').hexdigest(),
                        module_source=module, module_sha256=hashlib.sha256(module.encode()).hexdigest(),
                        directory_packages=[dict(name="main", path="test/a", files=["a.go"])],
                        adapted_sources={"a/a.go": "package main\n"}, native_compile=[compiled],
                        importcfg_sha256="config", expected_sha256="expected")
            report = dict(kind="language", go_version="go1.27.1", contract="compiledir", complete=True, outcomes=[case], shard=0, shards=1)
            path = root / "language.json"

            def assess(status):
                parity.save(path, report)
                self.assertEqual(parity.language_behavior([root])[0]["counts"], {status: 1})

            assess("passed")
            for obj, key, value in [(case, "directory_packages", []), (case, "native_compile", []),
                                    (case, "adapted_sources", {}), (case, "module_sha256", "other"),
                                    (case, "wrapper_source", "void main() {}"),
                                    (compiled, "exit", 1), (compiled, "timed_out", True),
                                    (compiled, "command", ["go", "build", "."]),
                                    (fixture, "files", []), (fixture, "recipe", "compiledir -N"),
                                    (case, "build_error", "error")]:
                present, saved = key in obj, obj.get(key)
                obj[key] = value
                assess("failed")
                if present:
                    obj[key] = saved
                else:
                    del obj[key]
            report["contract"] = case["contract"] = fixture["recipe"] = "rundir"
            assess("failed")
            case["native_link"] = dict(execution, command=["go", "tool", "link"])
            assess("passed")
            case["gart"]["output_sha256"] = "wrong"
            assess("failed")

    def test_child_buildrun_requires_original_source_and_real_child_execution(self):
        digest = lambda value: parity.hashlib.sha256(value.encode()).hexdigest()
        child = "package main\nfunc main() { panic(1) }\n"
        source = "// buildrun\npackage main\nconst prog = `" + child + "`\n"
        module = "module example.com/buildrunfixture\n\ngo 1.27.1\n"
        execution = dict(ran=True, exit=0, timed_out=False, output_sha256=digest(""))
        environment = {"PATH": "/proxy:/tools"}
        case = dict(
            fixture=dict(name="fixedbugs/issue46234", recipe="buildrun", adapter="child-buildrun",
                         files=[dict(path="fixedbugs/issue46234.go", sha256=digest(source))]),
            adapted_sources={"fixture.go": source}, generated_source=child,
            module_source=module, module_sha256=digest(module),
            wrapper_source="proxy", wrapper_sha256=digest("proxy"),
            expected_sha256=digest(""),
            native_compile=[dict(execution, command=["go", "build", "-gcflags=all=", "-o", "/native", "fixture.go"])],
            native=dict(execution, command=["/native"], environment=environment),
            gart=dict(execution, command=["dart", "/output/bin/main.dart"], environment=environment),
            native_child=dict(execution, exit=1, command=["/tools/go", "run", "/first/main.go"], output_sha256=digest("native panic")),
            gart_child=dict(execution, exit=2, command=["/tools/gart.exe", "run", "-frontend=source", "-format=false", "-o", "/child", "-runtime", "/runtime", "/second/main.go"], output_sha256=digest("gart panic")),
            native_diagnostics="native panic", gart_diagnostics="gart panic")
        self.assertTrue(parity.child_buildrun_valid(case, "go1.27.1", ""))
        for path, value in [
                (("gart_child",), None),
                (("gart_child", "ran"), False),
                (("gart_child", "timed_out"), True),
                (("gart_child", "exit"), 0),
                (("gart_child", "command"), ["go", "run", "/second/main.go"]),
                (("generated_source",), "package main"),
                (("adapted_sources", "fixture.go"), source + "// changed\n"),
                (("gart_diagnostics",), "different output"),
                (("native_compile",), []),
                (("fixture", "name"), "another-fixture"),
                (("wrapper_sha256",), "wrong")]:
            changed = copy.deepcopy(case)
            parent = changed
            for key in path[:-1]:
                parent = parent[key]
            parent[path[-1]] = value
            with self.subTest(path=path):
                self.assertFalse(parity.child_buildrun_valid(changed, "go1.27.1", ""))
        self.assertFalse(parity.child_buildrun_valid(case, "go1.27.1", "-N"))

    def test_directory_compiler_options_reach_all_packages(self):
        module = "module test\n\ngo 1.27.1\n"
        wrapper = 'import "package:test/a.dart";\nimport "package:test/b.dart";\nvoid main() {}\n'
        for contract, flag in (("compiledir", "-c=2"), ("compiledir", "-c=4"), ("rundir", "-l=4")):
            packages = [dict(name=name, path="test/" + name, files=[name + ".go"])
                        for name in ("a", "b")]
            if contract == "rundir":
                packages[-1]["name"] = "main"
            compiled = [dict(ran=True, exit=0, timed_out=False,
                             command=["go", "tool", "compile", "-e", flag, p["files"][0]])
                        for p in packages]
            case = dict(fixture=dict(name="p", recipe=contract + " " + flag, adapter="directory-packages",
                                     files=[dict(path="p.dir/" + p["files"][0]) for p in packages]),
                        contract=contract, directory_packages=packages, native_compile=compiled,
                        adapted_sources={"a/a.go": "package a\n", "b/b.go": "package b\n"},
                        module_source=module, module_sha256=parity.hashlib.sha256(module.encode()).hexdigest(),
                        importcfg_sha256="config", wrapper_source=wrapper,
                        wrapper_sha256=parity.hashlib.sha256(wrapper.encode()).hexdigest())
            if contract == "rundir":
                execution = dict(ran=True, exit=0, timed_out=False, output_sha256="expected")
                case.update(native_link=dict(execution, command=["go", "tool", "link"]),
                            native=dict(execution), gart=dict(execution), expected_sha256="expected")
            self.assertTrue(parity.directory_language_valid(case, "go1.27.1"))
            for execution in compiled:
                original = execution["command"]
                for flags in ([], [flag[:3] + "1"], [flag, flag[:3] + "8"]):
                    execution["command"] = original[:4] + flags + original[-1:]
                    self.assertFalse(parity.directory_language_valid(case, "go1.27.1"))
                execution["command"] = original
            case["fixture"]["recipe"] = contract + " " + flag[:3] + "8"
            self.assertFalse(parity.directory_language_valid(case, "go1.27.1"))

            case["fixture"]["recipe"] = contract + " " + flag
            case["contract"] = "compiledir" if contract == "rundir" else "rundir"
            self.assertFalse(parity.directory_language_valid(case, "go1.27.1"))

    def test_directory_linker_options_reach_native_link(self):
        module = "module test\n\ngo 1.27.1\n"
        execution = dict(ran=True, exit=0, timed_out=False, output_sha256="expected")
        command = ["go", "tool", "link", "-s", "-w", "-buildid=test", "-o", "native", "-importcfg=config", "-strictdups=2", "-w=0", "a.o"]
        case = dict(
            fixture=dict(name="p", recipe="rundir -ldflags -strictdups=2 -w=0", adapter="directory-packages", files=[dict(path="p.dir/a.go")]),
            contract="rundir", directory_packages=[dict(name="main", path="test/a", files=["a.go"])],
            native_compile=[dict(execution, command=["go", "tool", "compile", "-e", "a.go"])],
            adapted_sources={"a/a.go": "package main\n"},
            module_source=module, module_sha256=parity.hashlib.sha256(module.encode()).hexdigest(),
            importcfg_sha256="config", native_link=dict(execution, command=command),
            native=dict(execution), gart=dict(execution), expected_sha256="expected")
        self.assertTrue(parity.directory_language_valid(case, "go1.27.1"))
        for changed in [command[:9] + command[-1:], command[:-2] + ["-w=1", "a.o"],
                        command[:-1] + ["-strictdups=0", "a.o"], command[:-1] + ["other.o"],
                        ["go", "build", "."]]:
            case["native_link"]["command"] = changed
            self.assertFalse(parity.directory_language_valid(case, "go1.27.1"))
        case["native_link"]["command"] = command
        case["fixture"]["recipe"] += " -s=0"
        self.assertFalse(parity.directory_language_valid(case, "go1.27.1"))

    def test_separate_directory_packages_preserve_file_identity(self):
        module = "module test\n\ngo 1.27.1\n"
        wrapper = 'import "package:test/a.dart";\nimport "package:test/b.dart";\nvoid main() {}\n'
        for contract in ("compiledir", "errorcheckdir"):
            with self.subTest(contract=contract):
                packages = [dict(name="a", path="test/" + name, files=[name + ".go"])
                            for name in ("a", "b")]
                compiled = [dict(ran=True, exit=0, timed_out=False,
                                 command=["go", "tool", "compile", "-e", "-d=panic", p["files"][0]])
                            for p in packages]
                if contract == "errorcheckdir":
                    compiled[-1]["exit"] = 2
                case = dict(fixture=dict(name="p", recipe=contract + " -s", adapter="directory-packages",
                                         files=[dict(path="p.dir/" + p["files"][0]) for p in packages]),
                            contract=contract, directory_packages=packages, native_compile=compiled,
                            native=dict(compiled[-1]),
                            adapted_sources={"a/a.go": "package a\n", "b/b.go": "package a\n"},
                            module_source=module, module_sha256=parity.hashlib.sha256(module.encode()).hexdigest(),
                            importcfg_sha256="config", wrapper_source=wrapper,
                            wrapper_sha256=parity.hashlib.sha256(wrapper.encode()).hexdigest())
                self.assertTrue(parity.directory_language_valid(case, "go1.27.1"))
                case["fixture"]["recipe"] = contract
                self.assertFalse(parity.directory_language_valid(case, "go1.27.1"))
                case["fixture"]["recipe"] = contract + " -s"
                packages[0]["files"].append("b.go")
                self.assertFalse(parity.directory_language_valid(case, "go1.27.1"))

    def test_package_directory_unicode_paths_require_explicit_mapping(self):
        import hashlib
        module = "module test\n\ngo 1.27.1\n"
        wrapper = 'import "package:test/p0.dart";\nimport "bin/main.dart";\nvoid main() {}\n'
        execution = dict(ran=True, exit=0, timed_out=False)
        packages = [dict(name="Þfoo", path="test/Þfoo", module_path="test/p0", files=["Þfoo.go"]),
                    dict(name="main", path="test/p0", module_path="test/p1", files=["p0.go"])]
        case = dict(fixture=dict(name="unicode", recipe="compiledir", adapter="directory-packages",
                                 files=[dict(path="unicode.dir/" + p["files"][0]) for p in packages]),
                    contract="compiledir", directory_packages=packages,
                    native_compile=[dict(execution, command=["go", "tool", "compile", "-e", p["files"][0]]) for p in packages],
                    adapted_sources={"p0/Þfoo.go": "package Þfoo\n", "p1/p0.go": "package main\n"},
                    module_source=module, module_sha256=hashlib.sha256(module.encode()).hexdigest(),
                    importcfg_sha256="config", wrapper_source=wrapper,
                    wrapper_sha256=hashlib.sha256(wrapper.encode()).hexdigest())
        self.assertTrue(parity.directory_language_valid(case, "go1.27.1"))
        for package in packages:
            original = package.pop("module_path")
            self.assertFalse(parity.directory_language_valid(case, "go1.27.1"))
            package["module_path"] = "test/wrong"
            self.assertFalse(parity.directory_language_valid(case, "go1.27.1"))
            package["module_path"] = original
        packages[1]["module_path"] = "test/p0"
        self.assertFalse(parity.directory_language_valid(case, "go1.27.1"))

    def test_directory_recipe_requires_original_module_and_sources(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            fixture = dict(name="fixedbugs/p", recipe="runindir", adapter="directory-module",
                           module_go_version="1.14", files=[dict(path="fixedbugs/p.dir/main.go")])
            module = "module p.dir\ngo 1.14\n"
            execution = dict(ran=True, exit=0, timed_out=False, output_sha256="original",
                             command=["go", "run", "."], environment=dict(GOPATH="/fixture"))
            case = dict(fixture=fixture, contract="runindir", status="passed", native=dict(execution),
                        gart=dict(execution), expected_sha256="original", module_source=module,
                        module_sha256=parity.hashlib.sha256(module.encode()).hexdigest())
            report = dict(kind="language", complete=True, outcomes=[case], shard=0, shards=1)

            def assess(status):
                parity.save(root / "language.json", report)
                self.assertEqual(parity.language_behavior([root])[0]["counts"], {status: 1})

            assess("passed")
            for recipe in ["runindir -gomodversion 1.14", 'runindir -gomodversion "1.14"',
                           "runindir -gomodversion '1.14'", 'runindir -gomodversion ""']:
                fixture["recipe"] = recipe
                assess("passed")
            fixture["recipe"] = "runindir"
            for object_, key, value in [(case, "module_source", "module other\ngo 1.14\n"),
                                        (case, "module_sha256", "different"),
                                        (fixture, "module_go_version", "1.27"),
                                        (fixture, "recipe", "runindir -gomodversion 1.27"),
                                        (fixture, "recipe", 'runindir -gomodversion "1.14'),
                                        (fixture, "recipe", 'runindir -gomodversion 1.14 -goexperiment fieldtrack'),
                                        (fixture, "adapter", "missing"),
                                        (fixture, "files", []),
                                        (case["native"], "command", ["go", "run", "main.go"]),
                                        (case["gart"], "environment", dict(GOPATH="/other")),
                                        (case["gart"], "exit", 1),
                                        (case["gart"], "timed_out", True)]:
                saved = object_[key]
                object_[key] = value
                assess("failed")
                object_[key] = saved

    def test_language_host_tools_require_identical_sdk_inputs(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            env = dict(GOROOT="sdk", STDLIB_IMPORTCFG="importcfg")
            execution = dict(ran=True, exit=0, timed_out=False,
                             output_sha256="expected", environment=env)
            case = dict(fixture=dict(name="tool", host_tools=True), status="passed",
                        native=dict(execution), gart=dict(execution),
                        expected_sha256="expected", importcfg_sha256="input")
            report = dict(kind="language", complete=True, outcomes=[case], shard=0, shards=1)
            path = root / "language.json"
            parity.save(path, report)
            self.assertEqual(parity.language_behavior([root])[0]["counts"], {"passed": 1})
            for changed in ({}, dict(GOROOT="other", STDLIB_IMPORTCFG="importcfg"),
                            dict(GOROOT="sdk", STDLIB_IMPORTCFG="other")):
                case["gart"]["environment"] = changed
                parity.save(path, report)
                self.assertEqual(parity.language_behavior([root])[0]["counts"], {"failed": 1})
            case["gart"]["environment"] = env
            del case["importcfg_sha256"]
            parity.save(path, report)
            self.assertEqual(parity.language_behavior([root])[0]["counts"], {"failed": 1})

    def test_language_inventory_never_implies_an_execution_pass(self):
        cases = [dict(name="run", action="run", native_eligible=True, gart_eligible=True, adapter="single-file"),
                 dict(name="assembly", action="asmcheck", native_eligible=True, gart_eligible=True,
                      adapter="missing", exclusion=dict(classification="target-limit", reason="native assembly assertions")),
                 dict(name="compile", action="compile", native_eligible=True, gart_eligible=True, adapter="missing"),
                 dict(name="disabled", action="skip", native_eligible=True, gart_eligible=True, adapter="missing"),
                 dict(name="blocked", action="run", native_eligible=True, gart_eligible=True,
                      adapter="single-file", rejection=["unsupported"]),
                 dict(name="deep", action="run", native_eligible=True, gart_eligible=True,
                      adapter="single-file", exclusion=dict(classification="implementation-gap", reason="deep recursion")),
                 dict(name="platform", action="run", native_eligible=False, gart_eligible=True, adapter="single-file")]
        result = parity.language_assessment(dict(cases=cases))
        self.assertEqual(result["counts"], {"unmeasured": 1, "adapter-missing": 1, "known-rejection": 1,
                                           "implementation-gap": 1, "ineligible": 1, "upstream-skip": 1, "target-limit": 1})
        self.assertEqual(result["actions"], {"run": 4, "compile": 1, "skip": 1, "asmcheck": 1})

    def test_run_recipe_preserves_companion_sources_and_arguments(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            execution = dict(ran=True, exit=0, timed_out=False, output_sha256="expected")
            wrapper = "import 'bin/main.dart' as program;\nvoid main() { program.main([\"arg1\",\"arg2\"]); }\n"
            case = dict(fixture=dict(name="main", file="main.go", recipe="run ./extra.go arg1 arg2",
                                     adapter="run-recipe", run_files=["extra.go"], arguments=["arg1", "arg2"],
                                     files=[dict(path="main.go", sha256="a" * 64), dict(path="extra.go", sha256="b" * 64)]),
                        contract="run", status="passed", expected_sha256="expected",
                        native=dict(execution, command=["go", "run", "-gcflags=all=", "main.go", "extra.go", "arg1", "arg2"]),
                        gart=dict(execution, command=["node", "run.js", "arg1", "arg2"]),
                        wrapper_source=wrapper, wrapper_sha256=parity.hashlib.sha256(wrapper.encode()).hexdigest())
            path = root / "language.json"

            def assess(value, status, target="js"):
                parity.save(path, dict(kind="language", target=target, complete=True, outcomes=[value], shard=0, shards=1))
                self.assertEqual(parity.language_behavior([root])[0]["counts"], {status: 1})

            for target in ("vm", "aot", "js", "wasm"):
                assess(case, "passed", target)
            for mutate in [
                lambda c: c["fixture"].update(run_files=[]),
                lambda c: c["fixture"].update(arguments=[]),
                lambda c: c["fixture"].update(recipe="run -race"),
                lambda c: c["fixture"].update(recipe="run ../extra.go arg1 arg2"),
                lambda c: c["fixture"]["files"].pop(),
                lambda c: c["native"]["command"].remove("extra.go"),
                lambda c: c["gart"]["command"].pop(),
                lambda c: c.update(wrapper_source="void main() {}"),
                lambda c: c.pop("wrapper_sha256"),
                lambda c: c.update(build_error="compiler error"),
            ]:
                changed = copy.deepcopy(case)
                mutate(changed)
                assess(changed, "failed")

    def test_run_recipe_preserves_compiler_options(self):
        recipes = {
            "run -gcflags=all=-d=checkptr": ["-gcflags=all=-d=checkptr"],
            "run -tags=purego -gcflags=all=-d=checkptr": ["-tags=purego", "-gcflags=all=-d=checkptr"],
            "run -gcflags -l=4": ["-gcflags", "-l=4"],
            'run -gcflags="-l"': ["-gcflags=-l"],
            "run -gcflags='all=-N -l'": ["-gcflags=all=-N -l"],
            "run -gcflags=-d=converthash=qy": ["-gcflags=-d=converthash=qy"],
            "run -gcflags=-d=softfloat": ["-gcflags=-d=softfloat"],
            "run -gcflags=-d=ssa/check/on": ["-gcflags=-d=ssa/check/on"],
            "run -gcflags=-l=4": ["-gcflags=-l=4"],
        }
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            execution = dict(ran=True, exit=0, timed_out=False, output_sha256="expected")
            for recipe, flags in recipes.items():
                case = dict(fixture=dict(name="main", file="main.go", recipe=recipe, adapter="run-recipe",
                                         native_flags=flags, files=[dict(path="main.go", sha256="a" * 64)]),
                            contract="run", status="passed", expected_sha256="expected",
                            native=dict(execution, command=["go", "run", "-gcflags=all="] + flags + ["main.go"]),
                            gart=dict(execution, command=["dart", "run", "main.dart"]))
                tags = ["purego"] if "-tags=purego" in flags else []
                case["fixture"]["build_tags"] = tags
                case["gart_build_tags"] = list(tags)
                path = root / "language.json"

                def assess(value, status, target):
                    parity.save(path, dict(kind="language", target=target, complete=True, outcomes=[value], shard=0, shards=1))
                    self.assertEqual(parity.language_behavior([root])[0]["counts"], {status: 1})

                for target in ["vm", "aot", "js", "wasm"]:
                    assess(case, "passed", target)
                    for mutate in [
                        lambda c: c["fixture"].update(build_tags=["changed"]),
                        lambda c: c.update(gart_build_tags=["changed"]),
                        lambda c: c["fixture"].update(native_flags=[]),
                        lambda c: c["fixture"].update(recipe=recipe + " -race"),
                        lambda c: c["fixture"].update(recipe="run -gcflags=-unknown"),
                        lambda c: c["native"]["command"].pop(3),
                        lambda c: c["native"]["command"].insert(3, "-race"),
                        lambda c: c["fixture"].update(arguments=["unexpected"]),
                        lambda c: c["gart"].update(output_sha256="different"),
                    ]:
                        changed = copy.deepcopy(case)
                        mutate(changed)
                        assess(changed, "failed", target)

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


    def test_compile_package_option_requires_the_original_command(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            execution = dict(ran=True, exit=0, timed_out=False)
            command = ["go", "tool", "compile", "-e", "-p=p", "-importcfg=config", "-o", "native.o", "sdk/p.go"]
            outcome = dict(fixture=dict(name="p", file="p.go", recipe="compile -p=p"),
                           contract="compile", status="passed", wrapper_sha256="wrapper",
                           native=dict(execution, command=command), gart=dict(execution))
            report = dict(kind="language", complete=True, outcomes=[outcome], shard=0, shards=1)
            path = root / "language.json"

            def assess(status):
                parity.save(path, report)
                self.assertEqual(parity.language_behavior([root])[0]["counts"], {status: 1})

            assess("passed")
            for changed in [command[:4] + command[5:], command[:4] + ["-p=other"] + command[5:],
                            command + ["-p=other"], command[:-1] + ["other.go"]]:
                outcome["native"]["command"] = changed
                assess("failed")
            outcome["native"]["command"] = command
            outcome["fixture"]["recipe"] = "compile -p=main"
            assess("failed")
            command = command[:4] + ["-p=main"] + command[5:]
            outcome["native"]["command"] = command
            assess("passed")
            for changed in [command[:4] + command[5:], command[:4] + ["-p=p"] + command[5:],
                            command + ["-p=p"], command[:-1] + ["other.go"]]:
                outcome["native"]["command"] = changed
                assess("failed")
            outcome["native"]["command"] = command
            outcome["fixture"]["recipe"] = "compile -p=other"
            assess("failed")

    def test_compile_options_require_the_original_command(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            execution = dict(ran=True, exit=0, timed_out=False)
            for option in ["-N", "-B", "-c=2", "-c=4", "-l", "-d=ssa/check/on", "-N -d=softfloat",
                           "-d=checkptr", "-d=libfuzzer", "-d=ssa/check/seed", "-d=ssa/check/seed=1", "-dynlink"]:
                command = ["go", "tool", "compile", "-e", "-p=p", "-importcfg=config"] + option.split() + ["-o", "native.o", "sdk/p.go"]
                outcome = dict(fixture=dict(name="p", file="p.go", recipe="compile " + option),
                               contract="compile", status="passed", wrapper_sha256="wrapper",
                               native=dict(execution, command=command),
                               gart=dict(execution, command=["dart", "compile", "kernel", "compile.dart", "-o", "compile.dill"]))
                report = dict(kind="language", complete=True, outcomes=[outcome], shard=0, shards=1)
                path = root / "language.json"

                def assess(status):
                    parity.save(path, report)
                    self.assertEqual(parity.language_behavior([root])[0]["counts"], {status: 1})

                assess("passed")
                for changed in [command[:6] + command[7:], command[:6] + ["-other"] + command[7:],
                                command + ["-l"], command[:-1] + ["other.go"],
                                command[:4] + ["-p=main"] + command[5:]]:
                    outcome["native"]["command"] = changed
                    assess("failed")
                outcome["native"]["command"] = command
                outcome["fixture"]["recipe"] += " -l"
                assess("failed")
                outcome["fixture"]["recipe"] = "compile -c=8"
                assess("failed")
                outcome["fixture"]["recipe"] = "compile " + option
                outcome["gart"]["command"] = ["dart", "analyze", "compile.dart"]
                assess("failed")

    def test_compile_environment_reaches_both_compilers(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            path = root / "language.json"
            execution = dict(ran=True, exit=0, timed_out=False)
            for action, option, key, value in [("compile", "-godebug", "GODEBUG", "gotypesalias=1"),
                                               ("compile", "-goexperiment", "GOEXPERIMENT", "fieldtrack"),
                                               ("build", "-goexperiment", "GOEXPERIMENT", "regabi,regabiargs")]:
                for previous in ["", "existing=1"]:
                    environment = {key: previous + "," + value if previous else value}
                    command = ["go", "tool", "compile", "-e", "-p=p", "-importcfg=config", "-o", "native.o", "sdk/p.go"]
                    build_command = ["/tmp/gart.exe", "build", "-format=false", "-runtime", "/runtime", "-o", "/output", "."]
                    outcome = dict(fixture=dict(name="p", file="p.go", recipe=action + " " + option + " " + value),
                                   contract=action, status="passed", wrapper_sha256="wrapper",
                                   native=dict(execution, command=command, environment=environment),
                                   gart_build=dict(execution, command=build_command, environment=environment),
                                   gart=dict(execution, environment=environment,
                                             command=["dart", "compile", "kernel", "compile.dart", "-o", "compile.dill"]))
                    if action == "build":
                        command = ["go", "build", "-gcflags=all=", "-o", "native.o", "sdk/p.go"]
                        outcome["native"]["command"] = command
                        outcome["wrapper_source"] = "void main() {}\n"
                        outcome["wrapper_sha256"] = parity.hashlib.sha256(outcome["wrapper_source"].encode()).hexdigest()
                        outcome["gart"]["command"] = ["dart", "compile", "kernel", "bin/main.dart", "-o", "compile.dill"]
                    report = dict(kind="language", complete=True, outcomes=[outcome], shard=0, shards=1,
                                  environment={key: previous})

                    def assess(status):
                        parity.save(path, report)
                        self.assertEqual(parity.language_behavior([root])[0]["counts"], {status: 1})

                    assess("passed")
                    for process in ["native", "gart_build", "gart"]:
                        for changed in [{}, {key: value + ",override"}]:
                            outcome[process]["environment"] = changed
                            assess("failed")
                        outcome[process]["environment"] = environment
                    for field, changed in [("ran", False), ("exit", 1), ("timed_out", True),
                                           ("command", ["gart.exe", "version"])]:
                        original = outcome["gart_build"][field]
                        outcome["gart_build"][field] = changed
                        assess("failed")
                        outcome["gart_build"][field] = original
                    outcome["native"]["command"] = command[:6] + [option, value] + command[6:]
                    assess("failed")
                    outcome["native"]["command"] = command
                    del report["environment"][key]
                    assess("failed")
                    report["environment"][key] = previous
                    del outcome["gart_build"]["timed_out"]
                    assess("failed")
                    outcome["gart_build"]["timed_out"] = False
                    del outcome["gart_build"]
                    assess("failed")

    def test_rejection_requires_original_assertions_and_diagnostics(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            execution = dict(ran=True, exit=1, timed_out=False,
                             output_sha256=parity.hashlib.sha256(b"invalid source").hexdigest())
            match = dict(ran=True, exit=0, timed_out=False)
            outcome = dict(fixture=dict(name="p", recipe="errorcheck"), contract="reject", status="passed",
                           wrapper_sha256="original matcher", build_error="invalid source",
                           native=dict(execution), gart=dict(execution),
                           native_match=dict(match), gart_match=dict(match),
                           native_diagnostics="invalid source", gart_diagnostics="invalid source")
            report = dict(kind="language", complete=True, outcomes=[outcome], shard=0, shards=1)
            path = root / "language.json"
            parity.save(path, report)
            self.assertEqual(parity.language_behavior([root])[0]["counts"], {"passed": 1})
            for process in ("native", "gart", "native_match", "gart_match"):
                saved = dict(outcome[process])
                for field, value in [("ran", False), ("exit", 0 if process in ("native", "gart") else 1),
                                     ("timed_out", True)]:
                    outcome[process] = {**saved, field: value}
                    parity.save(path, report)
                    self.assertEqual(parity.language_behavior([root])[0]["counts"], {"failed": 1})
                outcome[process] = saved
            for key in ("wrapper_sha256", "build_error", "native_diagnostics", "gart_diagnostics"):
                saved = outcome[key]
                outcome[key] = ""
                parity.save(path, report)
                self.assertEqual(parity.language_behavior([root])[0]["counts"], {"failed": 1})
                outcome[key] = saved
            outcome["gart_diagnostics"] = "changed diagnostic"
            parity.save(path, report)
            self.assertEqual(parity.language_behavior([root])[0]["counts"], {"failed": 1})

    def test_build_recipe_requires_original_build_and_entry_point(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            wrapper = "void main() {}\n"
            execution = dict(ran=True, exit=0, timed_out=False)
            outcome = dict(fixture=dict(name="p", file="p.go", recipe="build"), contract="build", status="passed",
                           wrapper_sha256=parity.hashlib.sha256(wrapper.encode()).hexdigest(), wrapper_source=wrapper,
                           native={**execution, "command": ["go", "build", "-gcflags=all=", "-o", "native.o", "/sdk/test/p.go"]},
                           gart={**execution, "command": ["dart", "compile", "kernel", "bin/main.dart", "-o", "compile.dill"]})
            path = root / "language.json"
            def assess(case, status):
                parity.save(path, dict(kind="language", contract="build", complete=True,
                                       outcomes=[case], shard=0, shards=1))
                self.assertEqual(parity.language_behavior([root])[0]["counts"], {status: 1})
            assess(outcome, "passed")
            for mutate in [
                lambda c: c.update(wrapper_source="different entry"),
                lambda c: c["native"].update(exit=1),
                lambda c: c["gart"].update(timed_out=True),
                lambda c: c["native"]["command"].__setitem__(1, "run"),
                lambda c: c["native"]["command"].__setitem__(5, "/sdk/test/other.go"),
                lambda c: c["gart"]["command"].__setitem__(3, "unrelated.dart"),
                lambda c: c["fixture"].update(recipe="build -race"),
            ]:
                changed = copy.deepcopy(outcome)
                mutate(changed)
                assess(changed, "failed")

            flagged = copy.deepcopy(outcome)
            flagged["fixture"].update(recipe="build -gcflags=-l=4", native_flags=["-gcflags=-l=4"])
            flagged["native"]["command"].insert(3, "-gcflags=-l=4")
            assess(flagged, "passed")
            for mutate in [
                lambda c: c["native"]["command"].pop(3),
                lambda c: c["native"]["command"].__setitem__(3, "-gcflags=-l"),
                lambda c: c["native"]["command"].insert(4, "-gcflags=-l"),
                lambda c: c["fixture"].pop("native_flags"),
                lambda c: c["fixture"].update(recipe="build -gcflags=-l=4 -race"),
            ]:
                changed = copy.deepcopy(flagged)
                mutate(changed)
                assess(changed, "failed")

    def test_rejection_recipe_preserves_panic_flag(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            diagnostic = "invalid source"
            execution = dict(ran=True, exit=1, timed_out=False,
                             output_sha256=parity.hashlib.sha256(diagnostic.encode()).hexdigest())
            match = dict(ran=True, exit=0, timed_out=False)
            outcome = dict(fixture=dict(name="p", recipe="errorcheck -d=panic"), contract="reject", status="passed",
                           wrapper_sha256="original matcher", build_error=diagnostic,
                           native={**execution, "command": ["go", "tool", "compile", "-d=panic", "p.go"]},
                           gart=dict(execution), native_match=dict(match), gart_match=dict(match),
                           native_diagnostics=diagnostic, gart_diagnostics=diagnostic)
            report = dict(kind="language", complete=True, outcomes=[outcome], shard=0, shards=1)
            path = root / "language.json"
            for recipe, flag, status in [("errorcheck -d=panic", "-d=panic", "passed"),
                                          ("errorcheck -d=panic", "-e", "failed"),
                                          ("errorcheck -0 -d=panic", "-d=panic", "failed"),
                                          ("errorcheck -e", "-e", "passed"),
                                          ("errorcheck -e", "-d=panic", "failed"),
                                          ("errorcheck -e", "-e=0", "failed"),
                                          ("errorcheck -e", "-e=false", "failed"),
                                          ("errorcheck -e=0", "-e", "failed"),
                                          ("errorcheck -0 -e", "-e", "failed")]:
                outcome["fixture"]["recipe"] = recipe
                outcome["native"]["command"][3] = flag
                parity.save(path, report)
                self.assertEqual(parity.language_behavior([root])[0]["counts"], {status: 1})

            outcome["fixture"]["recipe"] = "errorcheck -e"
            outcome["native"]["command"] = ["go", "tool", "compile", "-e", "-e=0", "p.go"]
            parity.save(path, report)
            self.assertEqual(parity.language_behavior([root])[0]["counts"], {"failed": 1})

    def test_standard_library_rejection_requires_both_modes(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            diagnostic = "//go:unknown is not allowed in the standard library"
            execution = dict(ran=True, exit=1, timed_out=False,
                             output_sha256=parity.hashlib.sha256(diagnostic.encode()).hexdigest())
            match = dict(ran=True, exit=0, timed_out=False)
            outcome = dict(fixture=dict(name="p", recipe="errorcheck -std"), contract="reject", status="passed",
                           wrapper_sha256="original matcher", build_error=diagnostic, gart_standard_library=True,
                           native={**execution, "command": ["go", "tool", "compile", "-std", "p.go"]},
                           gart=dict(execution), native_match=dict(match), gart_match=dict(match),
                           native_diagnostics=diagnostic, gart_diagnostics=diagnostic)

            def assess(case, status):
                parity.save(root / "language.json", dict(kind="language", complete=True,
                            outcomes=[case], shard=0, shards=1))
                self.assertEqual(parity.language_behavior([root])[0]["counts"], {status: 1})

            assess(outcome, "passed")
            for mutate in [
                lambda c: c.pop("gart_standard_library"),
                lambda c: c.update(gart_standard_library=False),
                lambda c: c.update(gart_standard_library="true"),
                lambda c: c["native"]["command"].remove("-std"),
                lambda c: c["native"]["command"].insert(3, "-std"),
                lambda c: c["native"]["command"].insert(3, "-std=false"),
                lambda c: c["fixture"].update(recipe="errorcheck -std -race"),
                lambda c: c["fixture"].update(recipe="errorcheck"),
                lambda c: c.update(contract="diagnostics"),
            ]:
                changed = copy.deepcopy(outcome)
                mutate(changed)
                assess(changed, "failed")

    def test_compiler_rejection_requires_both_flags(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            diagnostic = "missing function body"
            execution = dict(ran=True, exit=1, timed_out=False,
                             output_sha256=parity.hashlib.sha256(diagnostic.encode()).hexdigest())
            match = dict(ran=True, exit=0, timed_out=False)
            outcome = dict(fixture=dict(name="fixedbugs/issue48097", file="fixedbugs/issue48097.go",
                                       recipe="errorcheck -complete"),
                           contract="reject", status="passed", wrapper_sha256="original matcher",
                           build_error=diagnostic, native_diagnostics=diagnostic, gart_diagnostics=diagnostic,
                           native_match=dict(match), gart_match=dict(match),
                           native={**execution, "exit": 2, "command": [
                               "go", "tool", "compile", "-p=p", "-d=panic", "-C", "-e",
                               "-importcfg=/tmp/importcfg", "-o", "native.o", "-complete", "/sdk/test/fixedbugs/issue48097.go"]},
                           gart={**execution, "command": [
                               "/tmp/gart.exe", "build", "-format=false", "-runtime", "/runtime",
                               "-o", "/output", "-complete", "/fixture/issue48097.go"]})

            def assess(case, status):
                parity.save(root / "language.json", dict(kind="language", complete=True,
                            outcomes=[case], shard=0, shards=1))
                self.assertEqual(parity.language_behavior([root])[0]["counts"], {status: 1})

            assess(outcome, "passed")
            for mutate in [
                lambda c: c["native"]["command"].__setitem__(10, "-complete=false"),
                lambda c: c["native"]["command"].insert(11, "-complete=false"),
                lambda c: c["native"]["command"].__setitem__(11, "/sdk/test/other.go"),
                lambda c: c["gart"].pop("command"),
                lambda c: c["gart"]["command"].__setitem__(1, "run"),
                lambda c: c["gart"]["command"].__setitem__(7, "-complete=false"),
                lambda c: c["gart"]["command"].insert(8, "-complete=false"),
                lambda c: c["gart"]["command"].__setitem__(8, "/fixture/other.go"),
                lambda c: c["gart"].update(timed_out=True),
                lambda c: c["fixture"].update(recipe="errorcheck -complete -e=0"),
            ]:
                changed = copy.deepcopy(outcome)
                mutate(changed)
                assess(changed, "failed")

            limited = copy.deepcopy(outcome)
            limited["fixture"]["recipe"] = "errorcheck -e=0"
            limited["native"]["command"][6] = "-e=0"
            limited["native"]["command"].pop(10)
            limited["gart"]["command"][7] = "-e=0"
            assess(limited, "passed")
            for mutate in [
                lambda c: c["native"]["command"].__setitem__(6, "-e"),
                lambda c: c["native"]["command"].insert(7, "-e"),
                lambda c: c["gart"]["command"].__setitem__(7, "-e"),
                lambda c: c["gart"]["command"].insert(8, "-e"),
                lambda c: c["gart"].pop("command"),
                lambda c: c["fixture"].update(recipe="errorcheck -e=0 -complete"),
            ]:
                changed = copy.deepcopy(limited)
                mutate(changed)
                assess(changed, "failed")

    def test_timed_rejection_requires_commands_scale_and_duration(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            diagnostic = "initialization cycle"
            execution = dict(ran=True, exit=1, timed_out=False, seconds=1,
                             environment={"GO_TEST_TIMEOUT_SCALE": "1"},
                             output_sha256=parity.hashlib.sha256(diagnostic.encode()).hexdigest())
            match = dict(ran=True, exit=0, timed_out=False)
            outcome = dict(fixture=dict(name="initexp", file="initexp.go", recipe="errorcheck -t 10"),
                           contract="reject", status="passed", wrapper_sha256="original matcher",
                           build_error=diagnostic, native_diagnostics=diagnostic, gart_diagnostics=diagnostic,
                           native_match=dict(match), gart_match=dict(match),
                           native={**copy.deepcopy(execution), "exit": 2, "command": [
                               "go", "tool", "compile", "-p=p", "-d=panic", "-C", "-e",
                               "-importcfg=/tmp/importcfg", "-o", "native.o", "/sdk/test/initexp.go"]},
                           gart={**copy.deepcopy(execution), "command": [
                               "/tmp/gart.exe", "build", "-format=false", "-runtime", "/runtime",
                               "-o", "/output", "/fixture/initexp.go"]})

            def assess(case, status):
                parity.save(root / "language.json", dict(kind="language", complete=True,
                            outcomes=[case], shard=0, shards=1))
                self.assertEqual(parity.language_behavior([root])[0]["counts"], {status: 1})

            assess(outcome, "passed")
            scaled = copy.deepcopy(outcome)
            for who in ("native", "gart"):
                scaled[who]["environment"]["GO_TEST_TIMEOUT_SCALE"] = "2"
                scaled[who]["seconds"] = 15
            assess(scaled, "passed")
            for mutate in [
                lambda c: c["native"]["command"].__setitem__(6, "-e=0"),
                lambda c: c["native"]["command"].__setitem__(10, "/sdk/test/other.go"),
                lambda c: c["gart"]["command"].__setitem__(1, "run"),
                lambda c: c["gart"]["command"].__setitem__(7, "/fixture/other.go"),
                lambda c: c["native"].pop("environment"),
                lambda c: c["gart"]["environment"].update(GO_TEST_TIMEOUT_SCALE="2"),
                lambda c: c["native"]["environment"].update(GO_TEST_TIMEOUT_SCALE="0"),
                lambda c: c["native"]["environment"].update(GO_TEST_TIMEOUT_SCALE="invalid"),
                lambda c: c["native"].update(seconds=10),
                lambda c: c["gart"].update(seconds=-1),
                lambda c: c["gart"].pop("seconds"),
                lambda c: c["gart"].update(seconds="1"),
                lambda c: c["gart"].update(timed_out=True),
                lambda c: c["fixture"].update(recipe="errorcheck -t 10 -e=0"),
            ]:
                changed = copy.deepcopy(outcome)
                mutate(changed)
                assess(changed, "failed")

    def test_rejection_recipe_preserves_language_version(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            diagnostic = "type parameter requires go1.18 or later"
            module = "module example.com/rejectfixture\n\ngo 1.17\n"
            execution = dict(ran=True, exit=1, timed_out=False,
                             output_sha256=parity.hashlib.sha256(diagnostic.encode()).hexdigest())
            match = dict(ran=True, exit=0, timed_out=False)
            outcome = dict(fixture=dict(name="p", recipe="errorcheck -lang=go1.17", module_go_version="1.17"),
                           contract="reject", status="passed", wrapper_sha256="original matcher", build_error=diagnostic,
                           module_source=module, module_sha256=parity.hashlib.sha256(module.encode()).hexdigest(),
                           native={**execution, "command": ["go", "tool", "compile", "-lang=go1.17", "p.go"]},
                           gart=dict(execution), native_match=dict(match), gart_match=dict(match),
                           native_diagnostics=diagnostic, gart_diagnostics=diagnostic)
            path = root / "language.json"
            def assess(case, status):
                parity.save(path, dict(kind="language", complete=True, outcomes=[case], shard=0, shards=1))
                self.assertEqual(parity.language_behavior([root])[0]["counts"], {status: 1})
            assess(outcome, "passed")
            for mutate in [
                lambda c: c["native"]["command"].__setitem__(3, "-lang=go1.18"),
                lambda c: c["fixture"].update(module_go_version="1.18"),
                lambda c: c.update(module_source=module.replace("1.17", "1.18")),
                lambda c: c.pop("module_sha256"),
                lambda c: c["fixture"].update(recipe="errorcheck -0 -lang=go1.17"),
            ]:
                changed = copy.deepcopy(outcome)
                mutate(changed)
                assess(changed, "failed")

    def test_successful_diagnostics_preserve_exit_matcher_and_language_version(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            diagnostic = ""
            module = "module example.com/rejectfixture\n\ngo 1.17\n"
            execution = dict(ran=True, exit=0, timed_out=False,
                             output_sha256=parity.hashlib.sha256(diagnostic.encode()).hexdigest())
            match = dict(ran=True, exit=0, timed_out=False)
            outcome = dict(fixture=dict(name="p", recipe="errorcheck -0 -lang=go1.17", module_go_version="1.17"),
                           contract="diagnostics", status="passed", wrapper_sha256="original matcher", build_error=diagnostic,
                           module_source=module, module_sha256=parity.hashlib.sha256(module.encode()).hexdigest(),
                           native={**execution, "command": ["go", "tool", "compile", "-lang=go1.17", "p.go"]},
                           gart=dict(execution), native_match=dict(match), gart_match=dict(match),
                           native_diagnostics=diagnostic, gart_diagnostics=diagnostic)
            path = root / "language.json"
            def assess(case, status):
                parity.save(path, dict(kind="language", complete=True, outcomes=[case], shard=0, shards=1))
                self.assertEqual(parity.language_behavior([root])[0]["counts"], {status: 1})
            assess(outcome, "passed")
            for mutate in [
                lambda c: c["native"]["command"].__setitem__(3, "-lang=go1.18"),
                lambda c: c["fixture"].update(module_go_version="1.18"),
                lambda c: c.update(module_source=module.replace("1.17", "1.18")),
                lambda c: c.pop("module_sha256"),
                lambda c: c["fixture"].update(recipe="errorcheck -lang=go1.17"),
                lambda c: c["native"].update(exit=1),
                lambda c: c["gart"].update(exit=1),
                lambda c: c["native_match"].update(exit=1),
                lambda c: c["gart_match"].update(ran=False),
                lambda c: c["gart"].update(timed_out=True),
                lambda c: c.pop("wrapper_sha256"),
                lambda c: c.update(build_error="unexpected error"),
                lambda c: c.update(native_diagnostics="unexpected diagnostic"),
            ]:
                changed = copy.deepcopy(outcome)
                mutate(changed)
                assess(changed, "failed")

    def test_generated_rejection_retains_exact_source_bytes(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source = b"package p\nvar \xc2A int\n"
            diagnostics = "invalid UTF-8 encoding"
            execution = dict(ran=True, exit=1, timed_out=False,
                             output_sha256=parity.hashlib.sha256(diagnostics.encode()).hexdigest())
            match = dict(ran=True, exit=0, timed_out=False)
            outcome = dict(fixture=dict(name="nul1", file="nul1.go", recipe="errorcheckoutput",
                                        files=[dict(path="nul1.go", sha256="a" * 64)]),
                           contract="errorcheckoutput", status="passed", wrapper_sha256="original matcher",
                           build_error=diagnostics, native=dict(execution), gart=dict(execution),
                           native_match=dict(match), gart_match=dict(match),
                           native_diagnostics=diagnostics, gart_diagnostics=diagnostics,
                           generator=dict(ran=True, exit=0, timed_out=False,
                                          command=["go", "run", "-gcflags=all=", "/sdk/test/nul1.go"],
                                          output_sha256=parity.hashlib.sha256(source).hexdigest()),
                           generated_bytes=parity.base64.b64encode(source).decode())
            path = root / "language.json"
            def assess(case, status):
                parity.save(path, dict(kind="language", contract="errorcheckoutput", complete=True,
                                       outcomes=[case], shard=0, shards=1))
                self.assertEqual(parity.language_behavior([root])[0]["counts"], {status: 1})
            assess(outcome, "passed")
            for mutate in [
                lambda c: c.pop("generated_bytes"),
                lambda c: c.update(generated_bytes="malformed!"),
                lambda c: c.update(generated_bytes=parity.base64.b64encode(source.decode(errors="replace").encode()).decode()),
                lambda c: c["generator"].update(ran=False),
                lambda c: c["generator"].update(exit=1),
                lambda c: c["generator"].update(timed_out=True),
                lambda c: c["generator"].update(command=["echo", "fake"]),
                lambda c: c["fixture"].update(recipe="runoutput"),
                lambda c: c["fixture"].update(recipe="errorcheckoutput -0"),
                lambda c: c["gart_match"].update(exit=1),
            ]:
                changed = copy.deepcopy(outcome)
                mutate(changed)
                assess(changed, "failed")

    def test_upstream_expected_failure_requires_observed_mismatches(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            text, mismatch = "invalid source", "original matcher mismatch"
            execution = dict(ran=True, exit=1, timed_out=False,
                             output_sha256=parity.hashlib.sha256(text.encode()).hexdigest())
            match = {**execution, "output_sha256": parity.hashlib.sha256(mismatch.encode()).hexdigest()}
            case = dict(fixture=dict(name="p", recipe="errorcheck", upstream_expected_failure=True),
                        contract="reject", status="upstream-expected-failure", wrapper_sha256="matcher",
                        build_error=text, native=dict(execution), gart=dict(execution),
                        native_match=dict(match), gart_match=dict(match),
                        native_diagnostics=text, gart_diagnostics=text,
                        native_mismatch=mismatch, gart_mismatch=mismatch)
            report = dict(kind="language", complete=True, outcomes=[case], shard=0, shards=1)
            path = root / "language.json"

            def assess(status):
                parity.save(path, report)
                self.assertEqual(parity.language_behavior([root])[0]["counts"], {status: 1})

            assess("upstream-expected-failure")
            for key in ("native", "gart", "native_match", "gart_match"):
                saved = dict(case[key])
                for field, value in (("ran", False), ("exit", 0), ("timed_out", True)):
                    case[key] = {**saved, field: value}
                    assess("failed")
                case[key] = saved
            for key in ("native_mismatch", "gart_mismatch"):
                case[key] = "different mismatch"
                assess("failed")
                case[key] = mismatch
            case["fixture"]["upstream_expected_failure"] = False
            assess("failed")
            case["fixture"]["upstream_expected_failure"] = True
            case["status"] = "passed"
            case["native_match"]["exit"] = case["gart_match"]["exit"] = 0
            assess("failed")

    def test_rejection_results_preserve_failures(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            execution = dict(ran=True, exit=1, timed_out=False,
                             output_sha256=parity.hashlib.sha256(b"invalid source").hexdigest())
            matched = dict(ran=True, exit=0, timed_out=False)
            case = dict(fixture=dict(name="p", recipe="errorcheck"), contract="reject", status="failed",
                        native=dict(execution), gart=dict(execution),
                        native_match=dict(matched), gart_match={**matched, "exit": 1},
                        native_diagnostics="invalid source", gart_diagnostics="invalid source")
            report = dict(kind="language", complete=True, outcomes=[case], shard=0, shards=1)
            path = root / "language.json"

            def assess(expected):
                parity.save(path, report)
                group = parity.language_behavior([root])[0]
                self.assertEqual(group["counts"], {"failed": 1})
                self.assertEqual(group["rejection_results"], {expected: 1})
                self.assertEqual(group["outcomes"][0]["rejection_result"], expected)

            assess("rejection-diagnostics-mismatch")
            case["gart"]["exit"] = 0
            assess("accepted-invalid-source")
            case["native_match"]["exit"] = 1
            assess("native-baseline-mismatch")
            case["native_match"]["exit"] = 0
            case["gart"]["timed_out"] = True
            assess("incomplete-evidence")
            case["gart"]["timed_out"] = False
            case["gart_diagnostics"] = "changed bytes"
            assess("incomplete-evidence")
            case["gart_diagnostics"] = "invalid source"
            case["gart"]["exit"] = 1
            parity.save(path, report)
            case["gart"]["exit"] = 0
            parity.save(root / "other.json", report)
            group = parity.language_behavior([root])[0]
            self.assertEqual(group["counts"], {"failed": 1})
            self.assertEqual(group["rejection_results"], {"conflicting": 1})

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
                dict(package="p", status="vm-suite"), dict(package="q", status="compile-blocked"),
                dict(package="r", status="runtime-blocked")])
            parity.save(manifests / "go1/os-arch.json", dict(source_sha256="sdk", packages=[
                dict(path=pkg, tests=[dict(name=name, eligible=True, kind="Test") for name in names])
                for pkg, names in [("p", ["TestPass", "TestPartial", "TestMissing"]), ("q", ["TestBlocked"]), ("r", ["TestRuntimeBlocked"])]]))
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
            self.assertEqual(groups[0]["unobserved_counts"], {"untested": 1, "blocked": 2})
            record["gart"]["exit"] = 1
            parity.save(reports / "failure.json", record)
            failed = parity.behavior([reports], manifests)[0]
            self.assertNotIn("passed", failed["top_level"])
            self.assertTrue(failed["problems"])
            record["short"] = False
            parity.save(reports / "full.json", record)
            self.assertEqual(len(parity.behavior([reports], manifests)), 2)

    def test_behavior_preserves_target_exclusions_without_assumed_passes(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            manifests = root / "manifests"
            reports = root / "reports"
            reports.mkdir()
            parity.save(manifests / "coverage-manifest.json", [dict(package="p", status="vm-suite")])
            parity.save(manifests / "go1/os-arch.json", dict(source_sha256="sdk", packages=[
                dict(path="p", tests=[dict(name=name, eligible=True, kind="Test")
                                     for name in ["TestPass", "TestHost", "TestMissing"]])]))
            record = dict(package="p", expected=["TestPass"], go_version="go1", goos="os", goarch="arch",
                          source_sha256="sdk", target="js", short=True, native=dict(exit=0), gart=dict(exit=0),
                          target_exclusions={"TestHost": "The browser has no local filesystem."},
                          comparison=dict(passed=["TestPass"], partial=[], excluded=[], problems=[]))
            parity.save(reports / "result.json", record)
            group = parity.behavior([reports], manifests)[0]
            self.assertEqual(group["top_level"], {"passed": 1, "excluded": 1})
            self.assertEqual(group["unobserved_counts"], {"untested": 1})
            excluded = next(x for x in group["outcomes"] if x["name"] == "TestHost")
            self.assertEqual(excluded["exclusion_reasons"], ["The browser has no local filesystem."])
            record["target_exclusions"]["TestHost"] = ""
            parity.save(reports / "result.json", record)
            with self.assertRaisesRegex(ValueError, "invalid target exclusion"):
                parity.behavior([reports], manifests)

    def test_behavior_combines_shards_only_with_matching_compiler_source(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            manifests = root / "manifests"
            parity.save(manifests / "coverage-manifest.json", [])
            paths = [root / str(i) for i in range(2)]
            for i, path in enumerate(paths):
                parity.save(path / "run.json", dict(source="same"))
                parity.save(path / "reports/result.json", dict(package="p", expected=[f"Test{i}"],
                    native=dict(exit=0), gart=dict(exit=0), comparison=dict(passed=[f"Test{i}"], problems=[])))
            groups = parity.behavior(paths, manifests)
            self.assertEqual(len(groups), 1)
            self.assertEqual(groups[0]["top_level"], {"passed": 2})
            self.assertEqual({r["manifest"] for r in groups[0]["reports"]},
                             {str(path / "run.json") for path in paths})
            parity.save(paths[1] / "run.json", dict(source="different"))
            self.assertEqual(len(parity.behavior(paths, manifests)), 2)
            for path in paths:
                parity.save(path / "run.json", {})
            self.assertEqual(len(parity.behavior(paths, manifests)), 2)
            parity.save(paths[1] / "reports/result.json", parity.read(paths[0] / "reports/result.json"))
            self.assertEqual(len(parity.behavior(paths, manifests)), 2)
            for path in paths:
                parity.save(path / "run.json", dict(source="same"))
            groups = parity.behavior(paths, manifests)
            self.assertEqual(len(groups), 1)
            self.assertEqual(len(groups[0]["reports"]), 1)
            parity.save(paths[1] / "run.json", dict(source="different"))
            self.assertEqual(len(parity.behavior(paths, manifests)), 2)

    def test_language_combines_shards_only_with_matching_compiler_source(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            paths = [root / str(i) for i in range(2)]
            for i, path in enumerate(paths):
                execution = dict(ran=True, exit=0, timed_out=False, output_sha256="original")
                outcome = dict(fixture=dict(name=f"p{i}", file=f"p{i}.go"), status="passed",
                               native=execution, gart=dict(execution), expected_sha256="original")
                parity.save(path / "run.json", dict(source="same"))
                parity.save(path / "reports/result.json", dict(kind="language", complete=True,
                            outcomes=[outcome], shard=i, shards=2))
            groups = parity.language_behavior(paths)
            self.assertEqual(len(groups), 1)
            self.assertEqual(groups[0]["counts"], {"passed": 2})
            self.assertEqual({r["manifest"] for r in groups[0]["reports"]},
                             {str(path / "run.json") for path in paths})
            parity.save(paths[1] / "run.json", dict(source="different"))
            self.assertEqual(len(parity.language_behavior(paths)), 2)
            for path in paths:
                parity.save(path / "run.json", {})
            self.assertEqual(len(parity.language_behavior(paths)), 2)
            parity.save(paths[1] / "reports/result.json", parity.read(paths[0] / "reports/result.json"))
            self.assertEqual(len(parity.language_behavior(paths)), 2)
            for path in paths:
                parity.save(path / "run.json", dict(source="same"))
            groups = parity.language_behavior(paths)
            self.assertEqual(len(groups), 1)
            self.assertEqual(len(groups[0]["reports"]), 1)
            parity.save(paths[1] / "run.json", dict(source="different"))
            self.assertEqual(len(parity.language_behavior(paths)), 2)

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
