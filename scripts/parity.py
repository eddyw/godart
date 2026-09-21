#!/usr/bin/env python3
"""Measure API contracts, retained upstream evidence, and application parity."""
import argparse
from collections import Counter, defaultdict
import hashlib
import json
import os
from pathlib import Path
import platform
import re
import signal
import subprocess
import sys
import time

ROOT = Path(__file__).resolve().parent.parent
MANIFESTS = ROOT / "internal/parity/testdata"


def read(path):
    return json.loads(path.read_text())


def save(path, data):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(data, indent=2, sort_keys=True) + "\n")


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def command(args, cwd, directory, name, timeout):
    started = time.monotonic()
    with (directory / (name + ".stdout")).open("wb") as stdout, (directory / (name + ".stderr")).open("wb") as stderr:
        process = subprocess.Popen(args, cwd=cwd, stdin=subprocess.DEVNULL,
                                   stdout=stdout, stderr=stderr, start_new_session=True)
        timed_out = False
        try:
            code = process.wait(timeout=timeout)
        except subprocess.TimeoutExpired:
            timed_out = True
            os.killpg(process.pid, signal.SIGKILL)
            process.wait()
            code = 124
    return dict(args=list(map(str, args)), exit=code, timeout=timed_out,
                seconds=round(time.monotonic() - started, 3),
                stdout=name + ".stdout", stderr=name + ".stderr")


def identity(root):
    result = {"directory": str(root)}
    head = subprocess.run(["git", "rev-parse", "HEAD"], cwd=root, capture_output=True, text=True)
    result["head"] = head.stdout.strip() if head.returncode == 0 else None
    names = subprocess.run(["git", "ls-files", "-z", "--cached", "--others", "--exclude-standard"],
                           cwd=root, capture_output=True)
    source_names = sorted(set(names.stdout.split(b"\0")) - {b""})
    if names.returncode != 0 or not source_names:
        result["head"] = None
        result["source_sha256"] = None
        return result
    h = hashlib.sha256()
    for name in source_names:
        path = root / os.fsdecode(name)
        h.update(name + b"\0")
        if path.is_symlink():
            h.update(os.fsencode(os.readlink(path)))
        elif path.is_file():
            h.update(path.read_bytes())
        else:
            h.update(b"deleted")
    result["source_sha256"] = h.hexdigest()
    return result


def provenance(path):
    for parent in path.parents:
        manifest = parent / "run.json"
        if manifest.is_file():
            data = read(manifest)
            return {"manifest": str(manifest), "compiler_snapshot": data.get("source")}
    return {"manifest": None, "compiler_snapshot": None, "unattributed_directory": str(path.parent)}


def certificate_input_problem(report):
    fixture = report.get("certificate_inputs")
    overlay = any(arg.startswith("-overlay=") for arg in report.get("native_args", []))
    if fixture is None:
        return "missing certificate input evidence" if overlay else None
    if not overlay:
        return "native compiler did not receive the certificate inputs"
    generator = fixture["generator"]
    if not generator["ran"] or generator["exit"] != 0 or generator["timed_out"]:
        return "certificate input generator failed"
    digest_text = lambda text: hashlib.sha256(text.encode()).hexdigest()
    if (digest_text(fixture["source"]) != fixture["source_sha256"] or
            digest_text(fixture["generator_source"]) != fixture.get("generator_sha256") or
            digest_text(fixture["output"]) != generator["output_sha256"]):
        return "certificate source or generator identity differs"
    try:
        values = json.loads(fixture["output"])
    except ValueError:
        return "invalid certificate input output"
    inputs = fixture["inputs"]
    if len(inputs) != 4 or values != [x["pem"] for x in inputs]:
        return "certificate inputs differ from generator output"
    source = fixture["source"].encode()
    end = len(source)
    for item in reversed(inputs):
        start, stop = item["start"], item["end"]
        if not 0 <= start < stop <= end or source[start:stop].decode() != item["expression"]:
            return "certificate setup positions differ"
        if not item["expression"].startswith("generatePEMCertWithRepeatSAN("):
            return "certificate adapter changed another expression"
        value = item["pem"]
        if not value.startswith("-----BEGIN CERTIFICATE-----\n") or not value.endswith("-----END CERTIFICATE-----\n"):
            return "certificate input has no PEM certificate"
        source = source[:start] + json.dumps(value).encode() + source[stop:]
        end = start
    if hashlib.sha256(source).hexdigest() != fixture["adapted_sha256"]:
        return "adapted certificate source differs"
    return None


def behavior(paths, manifests=MANIFESTS):
    groups = {}
    seen = set()
    for directory in paths:
        if not directory.is_dir():
            raise ValueError("report directory does not exist: " + str(directory))
        for path in sorted(directory.rglob("*.json")):
            if "work" in path.relative_to(directory).parts:
                continue
            report = read(path)
            if not isinstance(report, dict) or not {"comparison", "native", "gart", "expected", "package"} <= report.keys():
                continue
            stamp = digest(path)
            if stamp in seen:
                continue
            seen.add(stamp)
            config = {key: report.get(key) for key in (
                "go_version", "dart_version", "node_version", "goos", "goarch", "target", "short",
                "source_sha256", "build_tags", "gart_build_tags", "tool_tags", "cgo", "environment")}
            config.update(provenance(path))
            key = json.dumps(config, sort_keys=True)
            group = groups.setdefault(key, {"configuration": config, "reports": [], "outcomes": defaultdict(set),
                                            "reasons": defaultdict(set), "problems": []})
            comparison = report["comparison"]
            problems = list(comparison.get("problems") or [])
            input_problem = certificate_input_problem(report)
            if input_problem:
                problems.append(input_problem)
            if report["native"]["exit"] != 0 or report["gart"]["exit"] != 0:
                problems.append("a process failed")
            if not report["expected"]:
                problems.append("empty expected selection")
            outcomes = group["outcomes"]
            pkg = report["package"]
            for category in ("passed", "partial", "excluded"):
                for item in comparison.get(category) or []:
                    name = item["test"] if category == "excluded" else item
                    outcomes[pkg, name].add(category)
                    if category == "excluded":
                        group["reasons"][pkg, name].add(item["reason"])
            for name in report["expected"]:
                if problems:
                    outcomes[pkg, name].add("failed")
                elif (pkg, name) not in outcomes:
                    outcomes[pkg, name].add("unobserved")
            group["problems"].extend(f"{path}: {problem}" for problem in problems)
            group["reports"].append({"path": str(path), "sha256": stamp, "package": pkg,
                                     "filter": report.get("filter"), "problems": problems})
    result = []
    coverage = {x["package"]: x for x in read(manifests / "coverage-manifest.json")}
    for group in groups.values():
        outcomes = group.pop("outcomes")
        reasons = group.pop("reasons")
        records = []
        for (pkg, name), statuses in sorted(outcomes.items()):
            status = next(iter(statuses)) if len(statuses) == 1 else "conflicting"
            record = dict(package=pkg, name=name, status=status)
            if (pkg, name) in reasons:
                record["exclusion_reasons"] = sorted(reasons[pkg, name])
            records.append(record)
        group["top_level"] = dict(Counter(x["status"] for x in records if "/" not in x["name"]))
        group["subtests"] = dict(Counter(x["status"] for x in records if "/" in x["name"]))
        group["outcomes"] = records
        config = group["configuration"]
        inventory_path = manifests / str(config["go_version"]) / f'{config["goos"]}-{config["goarch"]}.json'
        group["inventory_status"] = "missing"
        if inventory_path.is_file():
            inventory = read(inventory_path)
            group["inventory_status"] = "match" if inventory["source_sha256"] == config["source_sha256"] else "different"
            missing = []
            for pkg in inventory["packages"]:
                for test in pkg["tests"] or []:
                    if test["eligible"] and test["kind"] != "Benchmark" and (pkg["path"], test["name"]) not in outcomes:
                        classification = coverage.get(pkg["path"], {}).get("status")
                        status = "blocked" if classification in ("compile-blocked", "runtime-blocked") else "untested"
                        missing.append(dict(package=pkg["path"], name=test["name"], status=status))
            group["unobserved_inventory"] = missing
            group["unobserved_counts"] = dict(Counter(x["status"] for x in missing))
        result.append(group)
    return result


def scope_assessment(api, path):
    review = read(path)
    entries = {entry["package"]: entry for entry in review["packages"]}
    current = review["go_version"] == api["go_version"]
    packages = []
    for name in api["unassessed_packages"]:
        entry = entries.get(name)
        status = "unassessed" if entry is None else "current" if current else "needs-review"
        packages.append({**(entry or {"package": name}), "review_status": status})
    return {"basis": review["basis"], "reviewed_go_version": review["go_version"],
            "counts": dict(Counter(entry["review_status"] for entry in packages)), "packages": packages}


def rejection_result(case, status):
    if status != "failed":
        return status
    for key in ("native", "gart", "native_match", "gart_match"):
        result = case.get(key, {})
        if not result.get("ran") or result.get("timed_out"):
            return "incomplete-evidence"
    for key in ("native", "gart"):
        diagnostics = case.get(key + "_diagnostics", "")
        if hashlib.sha256(diagnostics.encode()).hexdigest() != case[key].get("output_sha256"):
            return "incomplete-evidence"
    if case["native"]["exit"] <= 0 or case["native_match"]["exit"] != 0:
        return "native-baseline-mismatch"
    if case["gart"]["exit"] == 0:
        return "accepted-invalid-source"
    if case["gart"]["exit"] > 0 and case["gart_match"]["exit"] != 0:
        return "rejection-diagnostics-mismatch"
    return "incomplete-evidence"


def directory_language_valid(case, go_version):
    fixture = case["fixture"]
    contract = case["contract"]
    packages = case.get("directory_packages") or []
    adapted = case.get("adapted_sources") or {}
    compiled = case.get("native_compile") or []
    module = case.get("module_source", "")
    if (fixture.get("recipe") != contract or fixture.get("adapter") != "directory-packages"
            or not packages or len(compiled) != len(packages) or case.get("build_error")
            or not case.get("importcfg_sha256") or not go_version
            or module != "module test\n\ngo " + go_version.removeprefix("go") + "\n"
            or hashlib.sha256(module.encode()).hexdigest() != case.get("module_sha256")):
        return False
    original = {f["path"] for f in fixture.get("files", [])}
    expected = set()
    names = set()
    consumed = set()
    imports = []
    for package, execution in zip(packages, compiled):
        files = package.get("files") or []
        path = package.get("path", "")
        if (not files or not package.get("name") or package.get("name") in names or path != "test/" + files[0][:-3]
                or not execution.get("ran") or execution.get("exit") != 0 or execution.get("timed_out")):
            return False
        names.add(package.get("name"))
        for name in files:
            if not name.endswith(".go") or "/" in name or fixture["name"] + ".dir/" + name not in original:
                return False
            target_name = name + ".gart.go" if name.endswith("_test.go") else name
            expected.add(path.removeprefix("test/") + "/" + target_name)
            consumed.add(fixture["name"] + ".dir/" + name)
        library = "bin/main.dart" if package["name"] == "main" else "package:test/" + path.removeprefix("test/") + ".dart"
        imports.append("import " + json.dumps(library, ensure_ascii=False) + ";")
        command = execution.get("command", [])
        if command[:4] != ["go", "tool", "compile", "-e"] or [Path(p).name for p in command[-len(files):]] != files:
            return False
    prefix = fixture["name"] + ".dir/"
    source_files = {p for p in original if p.startswith(prefix) and "/" not in p[len(prefix):] and p.endswith(".go")}
    if consumed != source_files or set(adapted) != expected or any(not isinstance(s, str) or not s for s in adapted.values()):
        return False
    if contract == "compiledir":
        wrapper = "\n".join(imports) + "\nvoid main() {}\n"
        return (case.get("wrapper_source") == wrapper
                and case.get("wrapper_sha256") == hashlib.sha256(wrapper.encode()).hexdigest())
    link = case.get("native_link") or {}
    expected = case.get("expected_sha256")
    return (link.get("ran") and link.get("exit") == 0 and not link.get("timed_out")
            and link.get("command", [])[:3] == ["go", "tool", "link"]
            and packages[-1]["name"] == "main" and expected
            and all(case[k].get("output_sha256") == expected for k in ("native", "gart")))


def language_behavior(paths):
    groups = {}
    seen = set()
    for directory in paths:
        for path in sorted(directory.rglob("*.json")):
            if "work" in path.relative_to(directory).parts:
                continue
            report = read(path)
            if not isinstance(report, dict) or report.get("kind") != "language":
                continue
            stamp = digest(path)
            if stamp in seen:
                continue
            seen.add(stamp)
            config = {key: report.get(key) for key in (
                "go_version", "dart_version", "node_version", "goos", "goarch", "target", "source_sha256", "tool_tags", "cgo", "environment")}
            config.update(provenance(path))
            config["contract"] = report.get("contract") or next((c["contract"] for c in report["outcomes"] if c.get("contract")), "run")
            group = groups.setdefault(json.dumps(config, sort_keys=True),
                                      dict(configuration=config, reports=[], outcomes={}, problems=[]))
            local = set()
            for case in report["outcomes"]:
                name = case["fixture"]["name"]
                if name in local:
                    group["problems"].append(f"{path}: duplicate fixture {name}")
                    status = "failed"
                else:
                    status = case["status"] if report["complete"] else "unobserved"
                local.add(name)
                if status in {"passed", "upstream-expected-failure"}:
                    expected = case.get("expected_sha256")
                    executed = all(case[k]["ran"] and case[k]["exit"] == 0 and not case[k]["timed_out"]
                                   for k in ("native", "gart"))
                    if case.get("contract") in {"compiledir", "rundir"}:
                        valid = executed and directory_language_valid(case, report.get("go_version"))
                    elif case.get("contract") == "compile":
                        valid = (executed and case["fixture"]["recipe"] == "compile"
                                 and case.get("wrapper_sha256") and not case.get("build_error"))
                    elif case.get("contract") == "reject":
                        rejected = all(case[k]["ran"] and case[k]["exit"] > 0 and not case[k]["timed_out"]
                                       for k in ("native", "gart"))
                        upstream_failure = bool(case["fixture"].get("upstream_expected_failure"))
                        matcher_exit = 1 if upstream_failure else 0
                        matched = all(case.get(k, {}).get("ran") and case[k]["exit"] == matcher_exit
                                      and not case[k]["timed_out"] for k in ("native_match", "gart_match"))
                        if upstream_failure:
                            matched = matched and all(case.get(k + "_mismatch") and
                                hashlib.sha256(case[k + "_mismatch"].encode()).hexdigest() == case[k + "_match"].get("output_sha256")
                                for k in ("native", "gart"))
                        diagnostics = all(case.get(k + "_diagnostics") and
                                          hashlib.sha256(case[k + "_diagnostics"].encode()).hexdigest() == case[k]["output_sha256"]
                                          for k in ("native", "gart"))
                        valid = (rejected and matched and diagnostics and case.get("build_error")
                                 and case.get("wrapper_sha256") and case["fixture"]["recipe"] == "errorcheck"
                                 and (status == "upstream-expected-failure") == upstream_failure)
                    else:
                        valid = expected and executed and all(case[k]["output_sha256"] == expected
                                                              for k in ("native", "gart"))
                        if case.get("contract") == "runoutput":
                            generator = case.get("generator") or {}
                            source = case.get("generated_source")
                            valid = (valid and case["fixture"]["recipe"] == "runoutput" and source
                                     and generator.get("ran") and generator.get("exit") == 0
                                     and not generator.get("timed_out") and not case.get("build_error")
                                     and hashlib.sha256(source.encode()).hexdigest() == generator.get("output_sha256"))
                        elif case.get("contract") == "runindir":
                            fixture = case["fixture"]
                            recipe = re.fullmatch(r'''runindir(?:\s+-gomodversion\s+(?:"([^"]*)"|'([^']*)'|([^\s"']+)))?''', fixture.get("recipe", ""))
                            explicit_version = next((v for v in recipe.groups() if v), "") if recipe else ""
                            module = (f'module {Path(fixture["name"]).name}.dir\n'
                                      f'go {fixture.get("module_go_version", "")}\n')
                            prefix = fixture["name"] + ".dir/"
                            valid = (valid and recipe is not None
                                     and (not explicit_version or explicit_version == fixture.get("module_go_version"))
                                     and fixture.get("adapter") == "directory-module"
                                     and fixture.get("module_go_version")
                                     and case.get("module_source") == module
                                     and case.get("module_sha256") == hashlib.sha256(module.encode()).hexdigest()
                                     and any(f["path"].startswith(prefix) for f in fixture.get("files", []))
                                     and case["native"].get("command", [])[-1:] == ["."]
                                     and case["native"].get("environment", {}).get("GOPATH")
                                     and case["native"]["environment"] == case["gart"].get("environment")
                                     and not case.get("build_error"))
                    valid = valid and (status == "passed" or case.get("contract") == "reject")
                    valid = valid and config["contract"] in {"run", "runoutput", "runindir", "compiledir", "rundir", "compile", "reject"} and case.get("contract", "run") == config["contract"]

                    if case["fixture"].get("host_tools"):
                        native_env = case["native"].get("environment") or {}
                        gart_env = case["gart"].get("environment") or {}
                        valid = (valid and case.get("importcfg_sha256")
                                 and all(native_env.get(key) and native_env[key] == gart_env.get(key)
                                         for key in ("GOROOT", "STDLIB_IMPORTCFG")))
                    if not valid:
                        status = "failed"
                        group["problems"].append(f"{path}: invalid pass for {name}")
                record = group["outcomes"].setdefault(name, dict(name=name, fixture=case["fixture"], statuses=set()))
                record["statuses"].add(status)
                if config["contract"] == "reject":
                    record.setdefault("rejection_results", set()).add(rejection_result(case, status))
            group["reports"].append(dict(path=str(path), sha256=stamp, complete=report["complete"],
                                         shard=report["shard"], shards=report["shards"]))
    result = []
    for group in groups.values():
        records = []
        for name, record in sorted(group["outcomes"].items()):
            statuses = record.pop("statuses")
            if len(statuses) > 1:
                statuses.discard("unobserved")
            record["status"] = next(iter(statuses)) if len(statuses) == 1 else "conflicting"
            if "rejection_results" in record:
                causes = record.pop("rejection_results")
                if len(causes) > 1:
                    causes.discard("unobserved")
                record["rejection_result"] = next(iter(causes)) if len(causes) == 1 else "conflicting"
            records.append(record)
        group["outcomes"] = records
        group["counts"] = dict(Counter(x["status"] for x in records))
        if group["configuration"]["contract"] == "reject":
            group["rejection_results"] = dict(Counter(x["rejection_result"] for x in records))
        result.append(group)
    return result


def language_assessment(data):
    cases = data["cases"]
    def status(case):
        if not case["native_eligible"] or not case["gart_eligible"]:
            return "ineligible"
        if case.get("exclusion"):
            return case["exclusion"]["classification"]
        if case.get("rejection"):
            return "known-rejection"
        return "unmeasured" if case["adapter"] != "missing" else "adapter-missing"
    return {**data, "counts": dict(Counter(status(case) for case in cases)),
            "actions": dict(Counter(case["action"] for case in cases))}


def fuzz_behavior(paths):
    runs = []
    for directory in paths:
        for path in sorted(directory.rglob("fuzz.json")):
            report = read(path)
            if not isinstance(report, dict) or report.get("kind") != "fuzz-replay":
                continue
            problems = []
            status = report.get("status", "incomplete")
            if status == "passed":
                if not all(report.get(key) for key in ("package", "function", "go_version", "dart_version",
                                                       "goos", "goarch", "source_sha256", "fixture_sha256")):
                    problems.append("missing source or SDK identity")
                native = report.get("native", {})
                gart = report.get("gart", {})
                comparison = report.get("comparison", {})
                outcomes = native.get("outcomes", {})
                if (report.get("generation", {}).get("exit") != 0 or native.get("exit") != 0
                        or gart.get("exit") != 0):
                    problems.append("a process failed or lacks an exit result")
                if (len(outcomes) < 2 or any(value != "pass" for value in outcomes.values())
                        or outcomes != gart.get("outcomes")
                        or set(outcomes) != set(comparison.get("passed") or [])
                        or comparison.get("problems") or comparison.get("excluded") or comparison.get("partial")):
                    problems.append("replay lacks matching successful input identities")
                fixture_root = (path.parent / "fixture").resolve()
                files = report.get("fixtures") or []
                if not files:
                    problems.append("no retained fixture identities")
                for item in files:
                    fixture = (fixture_root / item["path"]).resolve()
                    if (not fixture.is_relative_to(fixture_root) or not fixture.is_file()
                            or digest(fixture) != item["sha256"]):
                        problems.append("missing or changed fixture: " + item["path"])
                if problems:
                    status = "failed"
            runs.append({"path": str(path), "sha256": digest(path), "provenance": provenance(path), **report,
                         "status": status, "evidence_problems": problems})
    return runs


def report(args):
    args.output.mkdir(parents=True, exist_ok=False)
    api = args.api
    if api is None:
        step = command(["go", "run", "./internal/surface", "-json"], ROOT, args.output, "api", args.timeout)
        if step["exit"]:
            save(args.output / "failure.json", step)
            return 1
        api = args.output / step["stdout"]
    data = read(api)
    language = args.language
    if language is None:
        step = command(["go", "run", "./internal/cmd/parity", "-action", "language"],
                       ROOT, args.output, "language", args.timeout)
        if step["exit"]:
            save(args.output / "failure.json", step)
            return 1
        language = args.output / step["stdout"]
    symbols = [x for p in data["packages"] for x in p["symbols"] or []]
    coverage = read(MANIFESTS / "coverage-manifest.json")
    omissions = [dict(package=pkg, test=name, **entry)
                 for pkg, tests in read(MANIFESTS / "omitted-tests.json").items()
                 for name, entry in tests.items()]
    applications = []
    for directory in args.applications:
        for path in sorted(directory.rglob("application.json")):
            applications.append({"path": str(path), **read(path)})
    result = {
        "schema": 1, "compiler": identity(ROOT),
        "limits": ["No whole-Go parity percentage exists. Reachability still needs review.",
                   "API matches compare declarations, not behavior. Differences need review.",
                   "Retained reports prove only their recorded source and configuration.",
                   "A blocked package does not exclude its portable tests.",
                   "Source scope reviews establish no API or behavioral passes. Test and subtest counts are separate."],
        "api": {"input": str(api.resolve()), "measurement": "retained-input" if args.api else "current-checkout",
                "sha256": digest(api), "counts": dict(Counter(x["status"] for x in symbols)), **data},
        "language": {"input": str(language.resolve()), "sha256": digest(language),
                     "measurement": "retained-input" if args.language else "current-checkout",
                     **language_assessment(read(language))},
        "coverage": {"packages": len(coverage), "counts": dict(Counter(x["status"] for x in coverage)), "entries": coverage},
        "scope": scope_assessment(data, MANIFESTS / "scope-assessment.json"),
        "omissions": {"counts": dict(Counter(x["classification"] for x in omissions)), "entries": omissions},
        "behavior": behavior(args.reports), "language_behavior": language_behavior(args.reports),
        "fuzz_replay": fuzz_behavior(args.reports), "applications": applications,
    }
    save(args.output / "report.json", result)
    lines = ["# Go parity report", "", "This report does not claim a percentage of whole-Go parity.", "",
             "API declarations: " + json.dumps(result["api"]["counts"], sort_keys=True),
             f"Unassessed standard packages: {len(data['unassessed_packages'])}",
             "Source scope reviews: " + json.dumps(result["scope"]["counts"], sort_keys=True),
             "Package classifications: " + json.dumps(result["coverage"]["counts"], sort_keys=True), ""]
    lines.extend(["Language recipes: " + json.dumps(result["language"]["counts"], sort_keys=True), ""])
    lines.extend(["Omitted assertions: " + json.dumps(result["omissions"]["counts"], sort_keys=True), ""])
    for replay in result["fuzz_replay"]:
        lines.append(f"Fuzz replay {replay['package']}/{replay['function']}: {replay['status']}. "
                     f"Retained new inputs: {replay.get('generated_corpus', 'unobserved')}.")
    for group in result["behavior"]:
        c = group["configuration"]
        lines.extend([f"## {c['go_version']} {c['goos']}/{c['goarch']} {c['target']} short={c['short']}", "",
                      f"Compiler snapshot: {c.get('compiler_snapshot') or 'unattributed'}",
                      "Top-level outcomes: " + json.dumps(group["top_level"], sort_keys=True),
                      "Subtest outcomes: " + json.dumps(group["subtests"], sort_keys=True),
                      "Unobserved inventory: " + json.dumps(group.get("unobserved_counts", {}), sort_keys=True),
                      f"SDK inventory: {group['inventory_status']}. Problems: {len(group['problems'])}", ""])
    for group in result["language_behavior"]:
        c = group["configuration"]
        lines.extend([f"Language {c['contract']} {c['go_version']} {c['goos']}/{c['goarch']} {c['target']}: "
                      + json.dumps(group["counts"], sort_keys=True),
                      f"Language evidence problems: {len(group['problems'])}", ""])
        if "rejection_results" in group:
            lines.extend(["Rejection results: " + json.dumps(group["rejection_results"], sort_keys=True), ""])
    for app in applications:
        lines.append(f"Application {app['project']['directory']}: {app['mode']} = {app['status']}")
    (args.output / "report.md").write_text("\n".join(lines) + "\n")
    print("\n".join(lines))
    return 0


def test_outcomes(native, gart):
    left = {}
    for line in native.splitlines():
        item = json.loads(line)
        if item.get("Test") and item.get("Action") in {"pass", "skip", "fail"}:
            name = item["Test"]
            if name in left:
                raise ValueError("duplicate native test: " + name)
            left[name] = item["Action"]
    right = {}
    for line in gart.splitlines():
        match = re.match(r"^\s*--- (PASS|SKIP|FAIL): (\S+) \([0-9.]+s\)$", line)
        if match:
            name = match[2]
            if name in right:
                raise ValueError("duplicate Gart test: " + name)
            right[name] = match[1].lower()
    return {"native": left, "gart": right, "equal": left == right,
            "complete": bool(left) and left == right and all(v == "pass" for v in left.values())}


def application(args, compiler=None):
    args.output.mkdir(parents=True, exist_ok=False)
    out = args.output.resolve()
    project = args.project.resolve()
    gart = str(args.gart.resolve())
    result = {"schema": 1, "project": identity(project), "compiler": compiler,
              "gart_sha256": digest(Path(gart)), "host": platform.platform(),
              "mode": args.mode, "pattern": args.pattern, "tags": args.tags, "status": "failed", "steps": []}
    tags = ["-tags=" + args.tags] if args.tags else []
    for tool, flags in [("go", ["version"]), ("dart", ["--version"])]:
        result[tool] = subprocess.check_output([tool] + flags, text=True, stderr=subprocess.STDOUT).strip()

    def execute(name, command_args, cwd=project):
        step = command(command_args, cwd, out, name, args.timeout)
        result["steps"].append(step)
        save(out / "application.json", result)
        return step["exit"] == 0

    if args.mode == "test":
        # One package keeps test identities unambiguous across both runners.
        native = execute("native", ["go", "test", "-json", "-count=1", "-short", *tags, args.pattern])
        target = execute("gart", [gart, "test", "-v", "-short", "-frontend=source", "-o", str(out / "dart"), *tags, args.pattern])
        comparison = test_outcomes((out / "native.stdout").read_text(), (out / "gart.stdout").read_text())
        result["comparison"] = comparison
        result["status"] = "passed" if native and target and comparison["complete"] else "failed"
        if native and target and comparison["equal"] and comparison["native"] and not comparison["complete"]:
            result["status"] = "partial"
    else:
        native = execute("native-build", ["go", "build", "-o", str(out / "native"), *tags, args.pattern])
        target = execute("gart-build", [gart, "build", "-format=false", "-o", str(out / "dart"), *tags, args.pattern])
        if target:
            target = execute("pub", ["dart", "pub", "get"], out / "dart")
        if target:
            target = execute("kernel", ["dart", "compile", "kernel", "-o", str(out / "program.dill"), "bin/main.dart"], out / "dart")
        if native and target:
            result["status"] = "compiled"
            if args.mode == "run":
                native = execute("native-run", [str(out / "native")])
                target = execute("gart-run", ["dart", str(out / "program.dill")])
                equal = all((out / ("native-run." + stream)).read_bytes() == (out / ("gart-run." + stream)).read_bytes()
                            for stream in ["stdout", "stderr"])
                result["status"] = "passed" if native and target and equal else "failed"
                result["output_equal"] = equal
    save(out / "application.json", result)
    print(f"{args.mode}: {result['status']} ({out / 'application.json'})")
    return 0 if result["status"] in {"compiled", "passed"} else 1


def application_shard(cases, index, count):
    if count < 1 or index < 0 or index >= count:
        raise ValueError("invalid application shard")
    return sorted(cases, key=lambda case: case["name"])[index::count]


def applications(args):
    cases = application_shard(read(ROOT / "testdata/applications/manifest.json"), args.shard_index, args.shard_count)
    args.output.mkdir(parents=True, exist_ok=False)
    binary = args.output / "gart"
    compiler = identity(ROOT)
    build = command(["go", "build", "-o", str(binary), "./cmd/gart"], ROOT,
                    args.output, "compiler", args.timeout)
    after = identity(ROOT)
    if not compiler["source_sha256"] or compiler["source_sha256"] != after["source_sha256"]:
        compiler = None
    result = {"compiler": compiler, "build": build, "applications": [],
              "shard": args.shard_index, "shards": args.shard_count}
    if build["exit"]:
        save(args.output / "suite.json", result)
        return 1
    status = 0
    for case in cases:
        out = args.output / case["name"]
        code = application(argparse.Namespace(
            output=out, project=ROOT / case["project"], gart=binary,
            pattern=case["pattern"], mode=case["mode"], tags=case["tags"], timeout=args.timeout), compiler)
        result["applications"].append({"name": case["name"], "exit": code, "result": str(out / "application.json")})
        status = max(status, code)
        save(args.output / "suite.json", result)
    return status


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="operation", required=True)
    summary = sub.add_parser("report")
    summary.add_argument("--api", type=Path)
    summary.add_argument("--language", type=Path)
    summary.add_argument("--reports", type=Path, action="append", default=[])
    summary.add_argument("--applications", type=Path, action="append", default=[])
    app = sub.add_parser("application")
    app.add_argument("--project", type=Path, required=True)
    app.add_argument("--gart", type=Path, default=ROOT / "build/gart")
    app.add_argument("--pattern", default=".")
    app.add_argument("--tags", default="", help="use identical Go build tags for both compilers")
    app.add_argument("--mode", choices=["compile", "run", "test"], default="compile")
    suite = sub.add_parser("applications", help="build Gart and compare all pinned application fixtures")
    suite.add_argument("--shard-index", type=int, default=0)
    suite.add_argument("--shard-count", type=int, default=1)
    for command_parser in (summary, app, suite):
        command_parser.add_argument("--output", type=Path, required=True)
        command_parser.add_argument("--timeout", type=float, default=300)
    args = parser.parse_args()
    args.output = args.output.resolve()
    if args.timeout <= 0:
        parser.error("timeout must be positive")
    if args.operation == "application" and ("..." in args.pattern or args.pattern.startswith("-")):
        parser.error("select one application package or main.go file")
    try:
        return {"report": report, "application": application, "applications": applications}[args.operation](args)
    except (OSError, ValueError, KeyError, subprocess.SubprocessError) as error:
        print(str(error), file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
