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
    if names.returncode != 0:
        result["source_sha256"] = None
        return result
    h = hashlib.sha256()
    for name in sorted(set(names.stdout.split(b"\0")) - {b""}):
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
                        status = "blocked" if classification == "compile-blocked" else "untested"
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
        "coverage": {"packages": len(coverage), "counts": dict(Counter(x["status"] for x in coverage)), "entries": coverage},
        "scope": scope_assessment(data, MANIFESTS / "scope-assessment.json"),
        "omissions": {"counts": dict(Counter(x["classification"] for x in omissions)), "entries": omissions},
        "behavior": behavior(args.reports), "applications": applications,
    }
    save(args.output / "report.json", result)
    lines = ["# Go parity report", "", "This report does not claim a percentage of whole-Go parity.", "",
             "API declarations: " + json.dumps(result["api"]["counts"], sort_keys=True),
             f"Unassessed standard packages: {len(data['unassessed_packages'])}",
             "Source scope reviews: " + json.dumps(result["scope"]["counts"], sort_keys=True),
             "Package classifications: " + json.dumps(result["coverage"]["counts"], sort_keys=True), ""]
    lines.extend(["Omitted assertions: " + json.dumps(result["omissions"]["counts"], sort_keys=True), ""])
    for group in result["behavior"]:
        c = group["configuration"]
        lines.extend([f"## {c['go_version']} {c['goos']}/{c['goarch']} {c['target']} short={c['short']}", "",
                      f"Compiler snapshot: {c.get('compiler_snapshot') or 'unattributed'}",
                      "Top-level outcomes: " + json.dumps(group["top_level"], sort_keys=True),
                      "Subtest outcomes: " + json.dumps(group["subtests"], sort_keys=True),
                      "Unobserved inventory: " + json.dumps(group.get("unobserved_counts", {}), sort_keys=True),
                      f"SDK inventory: {group['inventory_status']}. Problems: {len(group['problems'])}", ""])
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


def application(args):
    args.output.mkdir(parents=True, exist_ok=False)
    out = args.output.resolve()
    project = args.project.resolve()
    gart = str(args.gart.resolve())
    result = {"schema": 1, "project": identity(project), "compiler": identity(ROOT),
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


def applications(args):
    args.output.mkdir(parents=True, exist_ok=False)
    binary = args.output / "gart"
    build = command(["go", "build", "-o", str(binary), "./cmd/gart"], ROOT,
                    args.output, "compiler", args.timeout)
    result = {"compiler": identity(ROOT), "build": build, "applications": []}
    if build["exit"]:
        save(args.output / "suite.json", result)
        return 1
    status = 0
    for case in read(ROOT / "testdata/applications/manifest.json"):
        out = args.output / case["name"]
        code = application(argparse.Namespace(
            output=out, project=ROOT / case["project"], gart=binary,
            pattern=case["pattern"], mode=case["mode"], tags=case["tags"], timeout=args.timeout))
        result["applications"].append({"name": case["name"], "exit": code, "result": str(out / "application.json")})
        status = max(status, code)
        save(args.output / "suite.json", result)
    return status


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="operation", required=True)
    summary = sub.add_parser("report")
    summary.add_argument("--api", type=Path)
    summary.add_argument("--reports", type=Path, action="append", default=[])
    summary.add_argument("--applications", type=Path, action="append", default=[])
    app = sub.add_parser("application")
    app.add_argument("--project", type=Path, required=True)
    app.add_argument("--gart", type=Path, default=ROOT / "build/gart")
    app.add_argument("--pattern", default=".")
    app.add_argument("--tags", default="", help="use identical Go build tags for both compilers")
    app.add_argument("--mode", choices=["compile", "run", "test"], default="compile")
    suite = sub.add_parser("applications", help="build Gart and compare all pinned application fixtures")
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
