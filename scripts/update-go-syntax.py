#!/usr/bin/env python3
"""Copy the Go compiler parser and record its exact source origin."""
import argparse
import hashlib
import json
from pathlib import Path
import shutil


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("goroot", type=Path)
    args = parser.parse_args()
    source = args.goroot / "src/cmd/compile/internal/syntax"
    destination = Path(__file__).resolve().parent.parent / "internal/gosyntax"
    files = {p.name: p for p in source.glob("*.go") if not p.name.endswith("_test.go")}
    if "syntax.go" not in files:
        parser.error("the Go SDK has no compiler parser source")
    files["LICENSE"] = args.goroot / "LICENSE"
    destination.mkdir(exist_ok=True)
    for previous in destination.glob("*.go"):
        if not previous.name.endswith("_test.go") and previous.name not in files:
            previous.unlink()
    hashes = {}
    for name, path in sorted(files.items()):
        shutil.copyfile(path, destination / name)
        hashes[name] = hashlib.sha256(path.read_bytes()).hexdigest()
    origin = {
        "version": (args.goroot / "VERSION").read_text().splitlines()[0],
        "source": "src/cmd/compile/internal/syntax",
        "files": hashes,
    }
    (destination / "ORIGIN.json").write_text(json.dumps(origin, indent=2) + "\n")


if __name__ == "__main__":
    main()
