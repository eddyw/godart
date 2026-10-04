#!/usr/bin/env python3
"""Select complete driver tests for a CI shard."""
import argparse
import json
import os
from pathlib import Path
import re
import sys


def shards(names, durations, mode, timeouts):
    def weight(name):
        prefix = "driver/linux" if sys.platform == "linux" else "driver"
        return durations.get(f"{prefix}/{mode}/{name}",
                             durations.get(f"driver/{mode}/{name}",
                                           durations.get(f"{prefix}/short/{name}",
                                                         durations.get(f"driver/short/{name}", 1)))) or 1

    selected = [[] for _ in timeouts]
    loads = [0.0] * len(timeouts)
    for name in sorted(names, key=lambda name: (-weight(name), name)):
        index = min(range(len(timeouts)), key=lambda index: loads[index] / timeouts[index])
        selected[index].append(name)
        loads[index] += weight(name)
    return selected


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--shard-index", type=int, required=True)
    parser.add_argument("--shard-count", type=int, required=True)
    parser.add_argument("--mode", choices=["short", "full"], required=True)
    parser.add_argument("--exclude", required=True)
    args = parser.parse_args()
    if args.shard_count < 1 or not 0 <= args.shard_index < args.shard_count:
        parser.error("require shard-count >= 1 and 0 <= shard-index < shard-count")
    names = [line for line in sys.stdin.read().splitlines()
             if re.fullmatch(r"(?:Test|Example|Fuzz)\w*", line)
             and not re.fullmatch(args.exclude, line)]
    durations = json.loads(Path("internal/driver/testdata/ci-durations.json").read_text())
    matrix = json.loads(Path("scripts/ci-matrix.json").read_text())["include"]
    timeouts = [1] * args.shard_count
    for job in matrix:
        if job["suite"] == "driver" and job["shards"] == args.shard_count:
            timeouts[job["shard"]] = job.get("full_timeout", job["timeout"]) if args.mode == "full" else job["timeout"]
    selected = shards(names, durations, args.mode, timeouts)[args.shard_index]
    directory = os.environ.get("GART_PARITY_REPORT_DIR")
    if directory:
        path = Path(directory) / f"driver-selection-{args.shard_index}.json"
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(json.dumps(dict(mode=args.mode, shard=args.shard_index,
                                       shards=args.shard_count, available=names,
                                       selected=selected), indent=2) + "\n")
    print("^(" + "|".join(re.escape(name) for name in selected) + ")$" if selected else "^$")


if __name__ == "__main__":
    main()
