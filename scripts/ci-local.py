#!/usr/bin/env python3
"""Run the CI matrix locally and retain isolated attempts and parity reports."""
import argparse
import concurrent.futures
import hashlib
import io
import json
import os
from pathlib import Path
import platform
import signal
import subprocess
import tarfile
import time
import uuid

ROOT = Path(__file__).resolve().parent.parent


def source_archive(destination):
    names = subprocess.check_output(
        ["git", "ls-files", "-z", "--cached", "--others", "--exclude-standard"], cwd=ROOT
    ).split(b"\0")
    digest = hashlib.sha256()
    with tarfile.open(destination, "w") as archive:
        for raw in sorted(set(names) - {b""}):
            name = os.fsdecode(raw)
            path = ROOT / name
            if not path.exists() and not path.is_symlink():
                continue
            info = archive.gettarinfo(str(path), arcname=name)
            info.uid = info.gid = info.mtime = 0
            info.uname = info.gname = ""
            digest.update(raw + b"\0" + str(info.mode).encode() + b"\0")
            if info.isfile():
                data = path.read_bytes()
                digest.update(data)
                archive.addfile(info, io.BytesIO(data))
            elif info.issym():
                digest.update(os.fsencode(info.linkname))
                archive.addfile(info)
    return digest.hexdigest()


def write_json(path, value):
    temporary = path.with_suffix(".tmp")
    temporary.write_text(json.dumps(value, indent=2) + "\n")
    temporary.replace(path)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--platform", choices=["host", "linux"], default="host")
    parser.add_argument("--mode", choices=["short", "full"], default="short")
    parser.add_argument("--suite", action="append", help="select a suite, repeat for more suites")
    parser.add_argument("--shard", type=int, help="select 1 shard within the selected suites")
    parser.add_argument("--jobs", type=int, choices=[1, 2], default=1)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--resume", action="store_true", help="retry failed or absent jobs with identical inputs")
    parser.add_argument("--image", default="godart-ci:go1.27.0-dart3.13.3-node26.8.2")
    args = parser.parse_args()
    if args.platform == "host" and args.jobs != 1:
        parser.error("host suites share the checkout and require --jobs=1")
    matrix = json.loads((ROOT / "scripts/ci-matrix.json").read_text())["include"]
    jobs = [j for j in matrix if "runner" not in j]
    known = {j["suite"] for j in jobs}
    if args.suite and set(args.suite) - known:
        parser.error("unknown suite: " + ", ".join(sorted(set(args.suite) - known)))
    jobs = [j for j in jobs if (not args.suite or j["suite"] in args.suite)
            and (args.shard is None or j["shard"] == args.shard)]
    if not jobs:
        parser.error("the selection has no jobs")
    destination = args.output.resolve()
    if destination.is_relative_to(ROOT) and not destination.is_relative_to(ROOT / "build"):
        parser.error("place repository output under build/")
    destination.mkdir(parents=True, exist_ok=True)
    manifest = destination / "run.json"
    if manifest.exists() != args.resume:
        parser.error("use a new output directory or --resume with an existing run")
    archive = destination / "source.tar"
    fingerprint = source_archive(destination / "source.next.tar")
    image = None
    if args.platform == "linux":
        image = subprocess.check_output(
            ["docker", "image", "inspect", args.image, "--format", "{{.Id}}"], text=True
        ).strip()
    versions = {}
    if args.platform == "host":
        for tool, flags in [("go", ["version"]), ("dart", ["--version"]), ("node", ["--version"])]:
            versions[tool] = subprocess.check_output([tool] + flags, text=True, stderr=subprocess.STDOUT).strip()
    config = dict(toolchains=versions, source=fingerprint, platform=args.platform, host=platform.platform(),
                  mode=args.mode, image=image, jobs=jobs, parallel=args.jobs)
    if args.resume:
        saved = json.loads(manifest.read_text())
        if saved != config:
            parser.error("source, environment, or selection changed; use a new output directory")
        (destination / "source.next.tar").unlink()
    else:
        (destination / "source.next.tar").replace(archive)
        write_json(manifest, config)
    work = destination / "work"
    if args.platform == "linux":
        # Cache ownership must match the unprivileged test process, including GOPATH.
        subprocess.run(["docker", "run", "--rm", "--platform", "linux/amd64",
                        "-v", "godart-ci-cache:/cache", image, "sh", "-c",
                        "mkdir -p /cache/go /cache/gomod /cache/build /cache/pub && chown -R 65534:65534 /cache"], check=True)
    else:
        if os.geteuid() == 0:
            parser.error("host filesystem tests require an unprivileged user")
        if not args.resume:
            work.mkdir()
            subprocess.run(["tar", "-xf", str(archive), "-C", str(work)], check=True)
        subprocess.run(["make", "get"], cwd=work, check=True)

    def run(job):
        name = f"{job['suite']}-{job['shard']}"
        directory = destination / name
        directory.mkdir(exist_ok=True)
        previous = sorted(directory.glob("attempt-*/result.json"))
        if args.resume and previous:
            result = json.loads(previous[-1].read_text())
            if result["exit"] == 0:
                print("REUSE", name, flush=True)
                return result
        attempt = directory / f"attempt-{len(list(directory.glob('attempt-*'))) + 1:03d}"
        attempt.mkdir()
        report = attempt / "reports"
        report.mkdir()
        minutes = job.get("full_timeout", job["timeout"]) if args.mode == "full" else job["timeout"]
        environment = dict(GOMAXPROCS="2", GOFLAGS="-p=2", DART_SUPPRESS_ANALYTICS="true",
                           GART_UPSTREAM_MODE=args.mode, GART_TEST_TIMEOUT=f"{minutes}m")
        command = ["bash", "scripts/ci.sh", job["suite"], str(job["shard"]), str(job["shards"])]
        container = "godart-ci-" + uuid.uuid4().hex[:12]
        if args.platform == "linux":
            environment.update(HOME="/tmp/home", GOPATH="/cache/go", GOMODCACHE="/cache/gomod",
                               GOCACHE="/cache/build", PUB_CACHE="/cache/pub", GART_PARITY_REPORT_DIR="/tmp/reports")
            invocation = ["docker", "run", "--platform", "linux/amd64", "--cpus=2", "--user=65534:65534",
                          "--name", container, "-v", f"{archive}:/tmp/source.tar:ro", "-v", "godart-ci-cache:/cache"]
            for key, value in environment.items():
                invocation += ["-e", f"{key}={value}"]
            invocation += [image, "bash", "-c",
                           'set -e; mkdir -p /tmp/home /tmp/work /tmp/reports; tar -xf /tmp/source.tar -C /tmp/work; '
                           'cd /tmp/work; go version; dart --version; node --version; make get; exec "$@"', "ci-local"] + command
        else:
            environment["GART_PARITY_REPORT_DIR"] = str(report)
            invocation = command
        print("START", name, flush=True)
        start = time.monotonic()
        with (attempt / "ci.log").open("w") as log:
            try:
                process = subprocess.Popen(invocation, cwd=work if args.platform == "host" else ROOT, env=os.environ | environment,
                                           stdout=log, stderr=subprocess.STDOUT, start_new_session=True)
                code = process.wait(timeout=(minutes + 5) * 60)
            except subprocess.TimeoutExpired:
                os.killpg(process.pid, signal.SIGKILL)
                process.wait()
                code = 124
            finally:
                if args.platform == "linux":
                    subprocess.run(["docker", "stop", "-t", "5", container], stdout=log, stderr=log)
                    copied = subprocess.run(["docker", "cp", container + ":/tmp/reports/.", str(report)], stdout=log, stderr=log)
                    subprocess.run(["docker", "rm", container], stdout=log, stderr=log)
            if args.platform == "linux" and copied.returncode and code == 0:
                code = copied.returncode
        result = dict(suite=job["suite"], shard=job["shard"], shards=job["shards"], mode=args.mode,
                      exit=code, seconds=round(time.monotonic() - start, 3), attempt=str(attempt.relative_to(destination)))
        write_json(attempt / "result.json", result)
        print(json.dumps(result), flush=True)
        return result

    results = []
    with concurrent.futures.ThreadPoolExecutor(max_workers=args.jobs) as pool:
        for result in pool.map(run, jobs):
            results.append(result)
            write_json(destination / "results.json", results)
    return int(any(result["exit"] for result in results))


if __name__ == "__main__":
    raise SystemExit(main())
