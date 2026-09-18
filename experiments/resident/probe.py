#!/usr/bin/env python3
"""Compare complete Gart requests with source, cached kernel, and resident execution."""
import argparse
import hashlib
import http.server
import json
import os
from pathlib import Path
import platform
import re
import statistics
import subprocess
import threading
import time


ROOT = Path(__file__).resolve().parents[2]
PHASE = re.compile(r"gart: phase=(\S+) duration=([0-9.]+)s")
HTTP_CALL = '''res, err := http.Get("http://127.0.0.1:0/")
	if err != nil {
		panic(err)
	}
	res.Body.Close()'''


class Handler(http.server.BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def do_GET(self):
        self.server.requests += 1
        self.send_response(200)
        self.send_header("Content-Length", "0")
        self.send_header("Connection", "close")
        self.end_headers()

    def log_message(self, *_):
        pass


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def normalized(text):
    # Only the playground's pointer identity varies between fresh processes.
    return re.sub(r"(?<=456 )0x[0-9a-f]+", "<pointer>", text)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--trials", type=int, default=5)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    if args.trials < 1:
        parser.error("--trials must be positive")
    destination = args.output.resolve()
    if not destination.is_relative_to(ROOT / "build"):
        parser.error("place output under the repository build directory")
    destination.mkdir(parents=True, exist_ok=False)
    fixture = ROOT / "experiments/resident/playground"
    original = (fixture / "main.go").read_text()
    if original.count(HTTP_CALL) != 1 or original.count('greet("world")') != 1:
        raise SystemExit("the fixture no longer matches the documented transformations")
    env = dict(os.environ, GOMAXPROCS="4", GOFLAGS="-p=2")
    binary = destination / "gart"
    subprocess.run(["go", "build", "-o", str(binary), "./cmd/gart"], cwd=ROOT, env=env, check=True)
    server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), Handler)
    server.requests = 0
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    endpoint = f"http://127.0.0.1:{server.server_port}/"
    rows = []
    shutdowns = []
    expected_requests = 0

    def command(name, argv, cwd, command_env):
        start = time.perf_counter()
        result = subprocess.run(argv, cwd=cwd, env=command_env, text=True,
                                stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=120)
        seconds = time.perf_counter() - start
        (destination / f"{name}.log").write_text(result.stdout)
        if result.returncode:
            raise RuntimeError(f"{name}: exit {result.returncode}\n{result.stdout}")
        return seconds, result.stdout

    try:
        for workload in ["http", "no-http"]:
            source = destination / workload / "source"
            source.mkdir(parents=True)
            (source / "go.mod").write_bytes((fixture / "go.mod").read_bytes())
            baseline = original
            if workload == "no-http":
                baseline = baseline.replace('\t"net/http"\n', "").replace(HTTP_CALL, "")
            baseline = baseline.replace("http://127.0.0.1:0/", endpoint)
            edited = baseline.replace('greet("world")', 'greet("earth")')
            expected = {}
            for label, contents in [("original", baseline), ("edited", edited)]:
                (source / "main.go").write_text(contents)
                _, text = command(f"{workload}-native-{label}", ["go", "run", "."], source, env)
                expected[label] = normalized(text)
                expected_requests += workload == "http"
            if expected["original"] == expected["edited"]:
                raise RuntimeError("the source edit did not change native output")
            for trial in range(args.trials):
                modes = ["source", "kernel", "resident"]
                offset = trial % len(modes)
                modes = modes[offset:] + modes[:offset]
                if trial % 2:
                    modes.reverse()
                for mode in modes:
                    out = destination / workload / f"{trial}-{mode}"
                    temporary = out / "temp"
                    temporary.mkdir(parents=True)
                    run_env = dict(env, TMPDIR=str(temporary))
                    info = out / "resident.info"
                    (source / "main.go").write_text(baseline)
                    try:
                        for state in ["cold", "warm", "edit"]:
                            if state == "edit":
                                (source / "main.go").write_text(edited)
                            name = f"{workload}-{trial}-{mode}-{state}"
                            flags = ["-timings", "-format=false", "-o", str(out), "./..."]
                            start = time.perf_counter()
                            phases = {}
                            if mode == "kernel":
                                _, text = command(name, [str(binary), "run", *flags], source, run_env)
                                phases.update({k: float(v) for k, v in PHASE.findall(text)})
                                if ("kernel-compile" in phases) != (state != "warm"):
                                    raise RuntimeError(f"{name}: unexpected production cache result")
                                execution = sum(phases.get(k, 0) for k in ["kernel-cache", "kernel-compile", "execute"])
                                frontend = sum(phases[k] for k in ["load", "verify", "analyze", "emit", "write", "format", "source-maps"])
                                pub = phases["pub"]
                            else:
                                frontend, build_text = command(name + "-build", [str(binary), "build", *flags], source, run_env)
                                phases.update({k: float(v) for k, v in PHASE.findall(build_text)})
                                pub = 0
                                if state == "cold":
                                    pub, _ = command(name + "-pub", ["dart", "pub", "get"], out, run_env)
                                argv = ["dart", "run"]
                                if mode == "resident":
                                    argv += ["--resident", "--quiet", "--resident-compiler-info-file=" + str(info)]
                                execution, text = command(name, [*argv, "bin/main.dart"], out, run_env)
                            total = time.perf_counter() - start
                            actual = "".join(line for line in text.splitlines(keepends=True) if not PHASE.search(line))
                            if normalized(actual) != expected["edited" if state == "edit" else "original"]:
                                raise RuntimeError(f"{name}: native output mismatch\n{actual}")
                            expected_requests += workload == "http"
                            kernel_dir = out / ".gart/kernel" if mode == "kernel" else temporary
                            artifacts = {str(p.relative_to(out)): dict(sha256=digest(p), bytes=p.stat().st_size)
                                         for p in kernel_dir.rglob("*.dill")} if mode != "source" else {}
                            row = dict(workload=workload, trial=trial, mode=mode, state=state,
                                       seconds=total, frontend=frontend, pub=pub, dart=execution, load_average=os.getloadavg(),
                                       phases=phases, artifacts=artifacts, source_sha256=digest(source / "main.go"))
                            rows.append(row)
                            with (destination / "trials.jsonl").open("a") as stream:
                                stream.write(json.dumps(row) + "\n")
                            print(f"{name}: total={total:.3f}s dart={execution:.3f}s", flush=True)
                    finally:
                        if info.exists():
                            elapsed, text = command(f"{workload}-{trial}-shutdown",
                                                    ["dart", "compilation-server", "shutdown",
                                                     "--resident-compiler-info-file=" + str(info)], out, run_env)
                            if info.exists():
                                raise RuntimeError("resident compiler info file survived shutdown")
                            shutdowns.append(dict(workload=workload, trial=trial, seconds=elapsed, output=text))
        if server.requests != expected_requests:
            raise RuntimeError(f"HTTP request count {server.requests}, expected {expected_requests}")
    finally:
        server.shutdown()
        server.server_close()
        thread.join()

    medians = []
    for workload in ["http", "no-http"]:
        for mode in ["source", "kernel", "resident"]:
            for state in ["cold", "warm", "edit"]:
                group = [r for r in rows if (r["workload"], r["mode"], r["state"]) == (workload, mode, state)]
                medians.append(dict(workload=workload, mode=mode, state=state,
                                    **{key: statistics.median(r[key] for r in group)
                                       for key in ["seconds", "frontend", "pub", "dart"]}))
    report = dict(host=platform.platform(), trials=args.trials, modes="rotated and reversed between trials",
                  versions={tool: subprocess.check_output([tool, flag], text=True).strip()
                            for tool, flag in [("go", "version"), ("dart", "--version")]},
                  revision=subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip(),
                  binary_sha256=digest(binary), fixture_sha256=digest(fixture / "main.go"),
                  environment={key: env[key] for key in ["GOMAXPROCS", "GOFLAGS"]},
                  cold="fresh Gart output, resident process, and resident kernel cache; warm Go/Pub/filesystem caches",
                  fixture="playground with fixed clock and local empty HTTP response; only pointer identities normalized",
                  http_requests=server.requests, shutdowns=shutdowns, rows=rows, medians=medians)
    (destination / "results.json").write_text(json.dumps(report, indent=2) + "\n")
    print(json.dumps(medians, indent=2))


if __name__ == "__main__":
    main()
