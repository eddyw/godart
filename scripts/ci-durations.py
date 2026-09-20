#!/usr/bin/env python3
"""Update upstream shard weights from retained local CI results."""
import argparse
import hashlib
import json
from pathlib import Path
import re


CASE = re.compile(r"^    --- PASS: (TestTestsOfGoroot|TestSelectedUpstream)/(\S+) \(([0-9]+(?:\.[0-9]+)?)s\)$", re.MULTILINE)


def measurements(directory):
    metadata = json.loads((directory / "run.json").read_text())
    mode = metadata["mode"]
    if mode not in ("short", "full"):
        raise ValueError(f"unknown mode: {mode}")
    logs = sorted(directory.glob("stdlib-*/attempt-*/ci.log"))
    logs.extend(sorted(directory.glob("upstream-priority-*/attempt-*/ci.log")))
    if (directory / "ci.log").is_file():
        logs.append(directory / "ci.log")
    weights = {}
    evidence = []
    for path in logs:
        result_path = path.with_name("result.json")
        if not result_path.is_file():
            continue
        result = json.loads(result_path.read_text())
        if result.get("mode", mode) != mode:
            raise ValueError(f"mode differs: {path}")
        data = path.read_bytes()
        accepted = {}
        for suite, package, elapsed in CASE.findall(data.decode()):
            duration = float(elapsed)
            if duration <= 0:
                continue
            key = suite + "/" + ("short/" if mode == "short" else "") + package
            weights[key] = max(weights.get(key, 0), duration)
            accepted[key] = max(accepted.get(key, 0), duration)
        if accepted:
            evidence.append({"log": str(path), "sha256": hashlib.sha256(data).hexdigest(),
                             "exit": result["exit"], "durations": accepted})
    if not weights:
        raise ValueError(f"no completed upstream cases: {directory}")
    return weights, {"run": str(directory), "source": metadata["source"],
                     "image": metadata.get("image"), "mode": mode, "logs": evidence}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("runs", nargs="+", type=Path)
    parser.add_argument("--base", type=Path, default=Path("internal/driver/testdata/ci-durations.json"))
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--evidence", required=True, type=Path)
    args = parser.parse_args()
    weights = json.loads(args.base.read_text())
    measured = {}
    evidence = []
    for directory in args.runs:
        values, record = measurements(directory)
        evidence.append(record)
        for key, value in values.items():
            measured[key] = max(measured.get(key, 0), value)
    weights.update(measured)
    args.output.write_text(json.dumps(weights, indent=2, sort_keys=True) + "\n")
    args.evidence.write_text(json.dumps(evidence, indent=2) + "\n")
    print(f"Updated {len(measured)} upstream durations.")


if __name__ == "__main__":
    main()
