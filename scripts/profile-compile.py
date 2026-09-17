#!/usr/bin/env python3
"""Measure gart phases and separate Dart source compilation from kernel execution."""
import argparse
import json
import os
import platform
from pathlib import Path
import re
import shutil
import statistics
import subprocess
import time

ROOT = Path(__file__).resolve().parent.parent
PHASE = re.compile(r"gart: phase=(\S+) duration=([0-9.]+)s")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--program", default="hello", help="directory under testdata/programs")
    parser.add_argument("--trials", type=int, default=3)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    if args.trials < 1:
        parser.error("--trials must be positive")
    program = (ROOT / "testdata/programs" / args.program).resolve()
    if not program.is_relative_to(ROOT / "testdata/programs") or not (program / "main.go").is_file():
        parser.error("select an existing testdata/programs fixture")
    destination = args.output.resolve()
    if destination.is_relative_to(ROOT) and not destination.is_relative_to(ROOT / "build"):
        parser.error("place repository output under build/")
    destination.mkdir(parents=True, exist_ok=False)
    source = destination / "source"
    shutil.copytree(program, source, ignore=shutil.ignore_patterns("dartout", ".dart_tool"))
    if not (source / "go.mod").is_file():
        (source / "go.mod").write_text("module example.com/profile\n\ngo 1.27\n")
    results = []

    def run(name, command, cwd):
        start = time.monotonic()
        result = subprocess.run(command, cwd=cwd, text=True, capture_output=True)
        seconds = time.monotonic() - start
        (destination / (name + ".stdout")).write_text(result.stdout)
        (destination / (name + ".stderr")).write_text(result.stderr)
        result.check_returncode()
        row = dict(name=name, seconds=seconds, phases={phase: float(value) for phase, value in PHASE.findall(result.stderr)})
        results.append(row)
        print(json.dumps(row), flush=True)
        return result.stdout + "".join(line for line in result.stderr.splitlines(keepends=True) if not PHASE.search(line))

    subprocess.run(["go", "build", "-o", str(destination / "gart"), "./cmd/gart"], cwd=ROOT, check=True)
    for trial in range(args.trials):
        out = destination / f"trial-{trial}"
        command = [str(destination / "gart"), "run", "-timings", "-o", str(out), "./..."]
        cold = run(f"{trial}-cold", command, source)
        warm = run(f"{trial}-warm", command, source)
        kernel = out / "main.dill"
        run(f"{trial}-kernel-compile", ["dart", "compile", "kernel", "-o", str(kernel), "bin/main.dart"], out)
        compiled = run(f"{trial}-kernel-run", ["dart", str(kernel)], out)
        direct = run(f"{trial}-source-run", ["dart", "--packages=.dart_tool/package_config.json", "bin/main.dart"], out)
        if not (cold == warm == compiled == direct):
            raise SystemExit("program outputs differ; use a deterministic fixture")
    groups = {}
    for row in results:
        name = row["name"].split("-", 1)[1]
        groups.setdefault(name, []).append(row)
    medians = {}
    for name, rows in groups.items():
        medians[name] = dict(seconds=statistics.median(row["seconds"] for row in rows), phases={
            phase: statistics.median(row["phases"][phase] for row in rows) for phase in rows[0]["phases"]})
    report = dict(host=platform.platform(), environment={key: os.environ.get(key, "") for key in ["GOMAXPROCS", "GOFLAGS"]}, program=args.program, trials=args.trials, cache="fresh gart output for cold; warm Go and Pub caches",
                  versions={tool: subprocess.check_output([tool, flag], text=True).strip()
                            for tool, flag in [("go", "version"), ("dart", "--version")]},
                  results=results, medians=medians)
    (destination / "results.json").write_text(json.dumps(report, indent=2) + "\n")
    print(json.dumps(medians, indent=2))


if __name__ == "__main__":
    main()
