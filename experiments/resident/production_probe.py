#!/usr/bin/env python3
"""Compare production resident compilation with a previous Gart binary."""
import argparse
import hashlib
import http.server
import json
import os
from pathlib import Path
import platform
import statistics
import signal
import tempfile
import subprocess
import threading
import time

from probe import Handler, HTTP_CALL, PHASE, ROOT, normalized
from session import digest


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--baseline", type=Path, required=True)
    parser.add_argument("--trials", type=int, default=5)
    args = parser.parse_args()
    baseline_binary = args.baseline.resolve()
    destination = args.output.resolve()
    if args.trials < 1 or not destination.is_relative_to(ROOT / "build"):
        parser.error("use positive trials and a new output directory under build/")
    destination.mkdir(parents=True, exist_ok=False)
    env = dict(os.environ, GOMAXPROCS="4", GOFLAGS="-p=2")
    binary = destination / "gart"
    subprocess.run(["go", "build", "-o", str(binary), "./cmd/gart"], cwd=ROOT, env=env, check=True)
    fixture = ROOT / "experiments/resident/playground"
    original = (fixture / "main.go").read_text()
    if original.count(HTTP_CALL) != 1 or original.count('greet("world")') != 1:
        raise RuntimeError("the fixture no longer matches the documented transformations")
    server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), Handler)
    server.requests = 0
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    rows, shutdowns = [], []
    expected_requests = 0

    def command(name, argv, cwd):
        start = time.perf_counter()
        result = subprocess.run(argv, cwd=cwd, env=env, stdout=subprocess.PIPE,
                                stderr=subprocess.STDOUT, text=True, timeout=120)
        elapsed = time.perf_counter() - start
        (destination / (name + ".log")).write_text(result.stdout)
        if result.returncode:
            raise RuntimeError(f"{name}: exit {result.returncode}\n{result.stdout}")
        return result, elapsed

    try:
        for workload in ["http", "no-http"]:
            source = destination / workload / "source"
            source.mkdir(parents=True)
            (source / "go.mod").write_bytes((fixture / "go.mod").read_bytes())
            baseline = original.replace("http://127.0.0.1:0/", f"http://127.0.0.1:{server.server_port}/")
            if workload == "no-http":
                baseline = original.replace('\t"net/http"\n', "").replace(HTTP_CALL, "")
            edited = baseline.replace('greet("world")', 'greet("earth")')
            expected = {}
            for label, contents in [("original", baseline), ("edited", edited)]:
                (source / "main.go").write_text(contents)
                result, _ = command(f"{workload}-native-{label}", ["go", "run", "."], source)
                expected[label] = normalized(result.stdout)
                expected_requests += workload == "http"
            if expected["original"] == expected["edited"]:
                raise RuntimeError("the edit did not change native output")
            for trial in range(args.trials):
                modes = ["baseline", "resident"] if trial % 2 == 0 else ["resident", "baseline"]
                for mode in modes:
                    output = destination / workload / f"{trial}-{mode}"
                    temporary = tempfile.TemporaryDirectory(prefix="gart-bench-", dir="/tmp")
                    previous_env = env
                    env = dict(env, TMPDIR=str(Path(temporary.name).resolve()))
                    (source / "main.go").write_text(baseline)
                    try:
                        for state in ["cold", "warm", "edit"]:
                            if state == "edit":
                                path = source / "main.go"
                                stamp = path.stat()
                                path.write_text(edited)
                                os.utime(path, ns=(stamp.st_atime_ns, stamp.st_mtime_ns))
                            name = f"{workload}-{trial}-{mode}-{state}"
                            start = time.perf_counter()
                            selected = binary if mode == "resident" else baseline_binary
                            result, _ = command(name, [str(selected), "run", "-timings", "-format=false",
                                                       "-o", str(output), "./..."], source)
                            phases = {k: float(v) for k, v in PHASE.findall(result.stdout)}
                            if ("kernel-compile" in phases) != (state != "warm"):
                                raise RuntimeError(f"{name}: unexpected cache result")
                            if mode == "resident" and state != "warm":
                                want = "kernel-cold" if state == "cold" else "kernel-incremental"
                                if want not in phases:
                                    raise RuntimeError(f"{name}: expected {want}")
                            dart = sum(phases.get(k, 0) for k in ["kernel-cache", "kernel-compile", "execute"])
                            frontend = sum(phases[k] for k in ["load", "verify", "analyze", "emit", "write", "format", "source-maps", "pub"])
                            total = time.perf_counter() - start
                            actual = "".join(line for line in result.stdout.splitlines(keepends=True) if not PHASE.search(line))
                            if normalized(actual) != expected["edited" if state == "edit" else "original"]:
                                raise RuntimeError(f"{name}: native output mismatch\n{result.stdout}")
                            expected_requests += workload == "http"
                            row = dict(workload=workload, trial=trial, mode=mode, state=state, seconds=total,
                                       frontend_and_pub=frontend, dart=dart, phases=phases,
                                       load_average=os.getloadavg())
                            rows.append(row)
                            with (destination / "trials.jsonl").open("a") as stream:
                                stream.write(json.dumps(row) + "\n")
                            print(f"{name}: total={total:.3f}s dart={dart:.3f}s", flush=True)
                    finally:
                        start = time.perf_counter()
                        marker = str(Path("/tmp") / f"gart-{os.getuid()}" / hashlib.sha256(env["TMPDIR"].encode()).hexdigest()[:16])
                        process_rows = subprocess.check_output(["ps", "-axo", "pid,command"], text=True).splitlines()[1:]
                        owned = [int(parts[0]) for line in process_rows
                                 if len(parts := line.strip().split(None, 1)) == 2 and marker in parts[1] and "/service.dart " in parts[1]]
                        for pid in owned:
                            os.kill(pid, signal.SIGTERM)
                        deadline = time.monotonic() + 5
                        while owned:
                            live = {int(line.strip()) for line in subprocess.check_output(["ps", "-axo", "pid="], text=True).splitlines()}
                            if not live.intersection(owned):
                                break
                            if time.monotonic() > deadline:
                                raise RuntimeError("benchmark service did not terminate")
                            time.sleep(.02)
                        if mode == "resident":
                            if len(owned) != 1:
                                raise RuntimeError(f"expected 1 private service, got {owned}")
                            shutdowns.append(dict(workload=workload, trial=trial, seconds=time.perf_counter() - start))
                        temporary.cleanup()
                        env = previous_env
        if server.requests != expected_requests:
            raise RuntimeError(f"HTTP count {server.requests}, expected {expected_requests}")
    finally:
        server.shutdown()
        server.server_close()
        thread.join()
    medians = []
    for workload in ["http", "no-http"]:
        for state in ["cold", "warm", "edit"]:
            for mode in ["baseline", "resident"]:
                group = [r for r in rows if (r["workload"], r["state"], r["mode"]) == (workload, state, mode)]
                medians.append(dict(workload=workload, state=state, mode=mode,
                                    **{k: statistics.median(r[k] for r in group)
                                       for k in ["seconds", "frontend_and_pub", "dart"]}))
    report = dict(host=platform.platform(), trials=args.trials, order="alternating mode order",
                  versions={tool: subprocess.check_output([tool, flag], text=True).strip()
                            for tool, flag in [("go", "version"), ("dart", "--version")]},
                  working_tree_base=subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip(),
                  working_tree_dirty=bool(subprocess.check_output(["git", "status", "--porcelain"], cwd=ROOT, text=True)),
                  binaries={"baseline": digest(baseline_binary), "resident": digest(binary)},
                  files={str(p.relative_to(ROOT)): digest(p) for p in
                         [Path(__file__), ROOT / "internal/driver/resident.dart", ROOT / "internal/driver/resident.go",
                          ROOT / "internal/driver/kernel.go", ROOT / "internal/driver/output_lock.go",
                          ROOT / "internal/driver/driver.go", ROOT / "internal/driver/run.go", fixture / "main.go"]},
                  environment={key: env[key] for key in ["GOMAXPROCS", "GOFLAGS"]},
                  cold="fresh output, service, and compiler, warm Go/Pub/filesystem caches",
                  http_requests=server.requests, shutdowns=shutdowns, rows=rows, medians=medians)
    (destination / "results.json").write_text(json.dumps(report, indent=2) + "\n")
    print(json.dumps(medians, indent=2))


if __name__ == "__main__":
    main()
