"""Isolated, serial resident compilation sessions for the Gart prototype."""
import hashlib
import json
import os
from pathlib import Path
import selectors
import shutil
import subprocess
import tempfile
import time
import urllib.parse
import uuid


class CompileError(RuntimeError):
    pass


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def file_uri(uri):
    parsed = urllib.parse.urlsplit(uri)
    if parsed.scheme != "file" or parsed.netloc not in ("", "localhost"):
        raise CompileError(f"unsupported source URI: {uri}")
    return Path(urllib.parse.unquote(parsed.path)).resolve()


class Session:
    """Own 1 compiler child. Call compile serially and close the session after use."""

    def __init__(self, output, sdk=None, defines=None):
        self.output = Path(output).resolve()
        self.sdk = Path(sdk).resolve() if sdk else Path(shutil.which("dart")).resolve().parent.parent
        self.defines = {"dart.vm.product": "false", **(defines or {})}
        self.directory = self.output / ".gart/resident-prototype"
        self.directory.mkdir(parents=True, exist_ok=True)
        self.process = None
        self.scratch = None
        self.selector = None
        self.buffer = b""
        self.identity = None
        self.sources = {}
        self.dependencies = set()
        self.last = {}

    def __enter__(self):
        return self

    def __exit__(self, *_):
        self.close()

    def close(self):
        if self.process:
            if self.process.poll() is None:
                try:
                    self.process.stdin.write(b"quit\n")
                    self.process.stdin.flush()
                    self.process.wait(timeout=3)
                except (BrokenPipeError, subprocess.TimeoutExpired):
                    self.process.kill()
                    self.process.wait()
            self.process.stdin.close()
            self.process.stdout.close()
            self.selector.close()
            self.process = None
            self.selector = None
        if self.scratch:
            self.scratch.cleanup()
            self.scratch = None
        self.buffer = b""
        self.identity = None
        self.sources = {}
        self.dependencies = set()

    def _inputs(self, entry):
        if (self.sdk / "version").read_text().strip() != "3.13.3":
            raise CompileError("the resident prototype requires Dart 3.13.3")
        sdk_files = ["version", "revision", "bin/dart", "bin/dartvm", "bin/dartaotruntime",
                     "bin/snapshots/frontend_server_aot.dart.snapshot",
                     "lib/_internal/vm_platform_strong.dill", "lib/libraries.json"]
        sdk_hashes = {name: digest(self.sdk / name) for name in sdk_files if (self.sdk / name).is_file()}
        config = self.output / ".dart_tool/package_config.json"
        data = config.read_bytes()
        parsed = json.loads(data)
        if parsed.get("configVersion") != 2 or not isinstance(parsed.get("packages"), list):
            raise CompileError("unsupported Dart package configuration")
        roots = {self.output}
        for package in parsed["packages"]:
            root = urllib.parse.urljoin(config.as_uri(), package["rootUri"])
            roots.add(file_uri(urllib.parse.urljoin(root.rstrip("/") + "/", package.get("packageUri", ""))))
        paths = {entry}
        for root in sorted(roots):
            for path in root.rglob("*.dart"):
                if any(part.startswith(".") for part in path.relative_to(root).parts):
                    continue
                if path.is_file():
                    paths.add(path.resolve())
        sources = {str(path): digest(path) for path in sorted(paths)}
        identity = dict(schema=1, sdk=str(self.sdk), sdk_files=sdk_hashes,
                        entry=str(entry), packages=hashlib.sha256(data).hexdigest(),
                        defines=self.defines, prototype=digest(Path(__file__)),
                        flags=["--target=vm", "--incremental", "--enable-asserts"])
        key = hashlib.sha256(json.dumps(identity, sort_keys=True).encode()).hexdigest()
        return key, sources

    def _start(self, identity):
        self.close()
        self.scratch = tempfile.TemporaryDirectory(prefix="compiler-", dir=self.directory)
        output = Path(self.scratch.name) / "program.dill"
        args = [str(self.sdk / "bin/dartaotruntime"),
                str(self.sdk / "bin/snapshots/frontend_server_aot.dart.snapshot"),
                "--sdk-root", str(self.sdk), "--platform", (self.sdk / "lib/_internal/vm_platform_strong.dill").as_uri(),
                "--target=vm", "--incremental", "--enable-asserts",
                "--packages", str(self.output / ".dart_tool/package_config.json"),
                "--output-dill", str(output)]
        args += [f"-D{k}={v}" for k, v in sorted(self.defines.items())]
        self.process = subprocess.Popen(args, cwd=self.output, stdin=subprocess.PIPE,
                                        stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
        self.selector = selectors.DefaultSelector()
        self.selector.register(self.process.stdout, selectors.EVENT_READ)
        self.identity = identity

    def _send(self, text):
        try:
            self.process.stdin.write(text.encode())
            self.process.stdin.flush()
        except BrokenPipeError as error:
            raise CompileError("resident compiler exited before the request") from error

    def _line(self, deadline):
        while b"\n" not in self.buffer:
            remaining = deadline - time.monotonic()
            if remaining <= 0 or not self.selector.select(remaining):
                raise CompileError("resident compilation exceeded 120 seconds")
            block = os.read(self.process.stdout.fileno(), 65536)
            if not block:
                raise CompileError("resident compiler exited before a complete response")
            self.buffer += block
        line, self.buffer = self.buffer.split(b"\n", 1)
        return line.decode().rstrip("\r")

    def _response(self):
        deadline = time.monotonic() + 120
        diagnostics = []
        line = self._line(deadline)
        while not line.startswith("result "):
            diagnostics.append(line)
            line = self._line(deadline)
        boundary = line.removeprefix("result ")
        line = self._line(deadline)
        while line != boundary:
            diagnostics.append(line)
            line = self._line(deadline)
        line = self._line(deadline)
        while not line.startswith(boundary + " "):
            if not line.startswith(("+file:", "-file:")):
                raise CompileError(f"unsupported dependency response: {line}")
            path = str(file_uri(line[1:]))
            if line[0] == "+":
                self.dependencies.add(path)
            else:
                self.dependencies.discard(path)
            line = self._line(deadline)
        path, errors = line.removeprefix(boundary + " ").rsplit(" ", 1)
        if int(errors):
            raise CompileError("\n".join(diagnostics))
        result = Path(path).resolve()
        if not result.is_relative_to(Path(self.scratch.name)):
            raise CompileError("resident compiler returned an unexpected artifact path")
        return result

    def compile(self, entry):
        start = time.perf_counter()
        entry = Path(entry).resolve()
        kernel = self.directory / (hashlib.sha256(str(entry).encode()).hexdigest() + ".dill")
        receipt = kernel.with_suffix(".json")
        identity, sources = self._inputs(entry)
        try:
            saved = json.loads(receipt.read_text())
        except (OSError, ValueError):
            saved = None
        if (isinstance(saved, dict) and saved.get("identity") == identity and saved.get("sources") == sources
                and kernel.is_file() and saved.get("kernel") == digest(kernel)):
            self.last = dict(mode="hit", seconds=time.perf_counter() - start, invalidated=[])
            return kernel
        inputs_time = time.perf_counter() - start
        cold = self.identity != identity or not self.process or self.process.poll() is not None
        changed = sorted(name for name in sources.keys() | self.sources.keys() if sources.get(name) != self.sources.get(name))
        compile_start = time.perf_counter()
        try:
            if cold:
                self._start(identity)
                self._send("compile " + entry.as_uri() + "\n")
            else:
                boundary = uuid.uuid4().hex
                self._send("recompile " + boundary + "\n" +
                           "".join(Path(name).as_uri() + "\n" for name in changed) + boundary + "\n")
            result = self._response()
            if not self.dependencies.issubset(sources):
                missing = sorted(self.dependencies - sources.keys())
                raise CompileError("dependencies outside the source snapshot: " + ", ".join(missing))
            if self._inputs(entry) != (identity, sources):
                raise CompileError("compiler inputs changed during compilation; retry the request")
            compile_time = time.perf_counter() - compile_start
            self._send("accept\nreset\n")
            self.sources = sources
            with tempfile.NamedTemporaryFile(dir=self.directory, delete=False) as temporary:
                pending = Path(temporary.name)
            try:
                shutil.copyfile(result, pending)
                artifact_hash = digest(pending)
                os.replace(pending, kernel)
                pending.write_text(json.dumps(dict(identity=identity, sources=sources, kernel=artifact_hash), sort_keys=True))
                os.replace(pending, receipt)
            finally:
                pending.unlink(missing_ok=True)
            self.last = dict(mode="cold" if cold else "incremental", seconds=time.perf_counter() - start,
                             inputs=inputs_time, compile=compile_time, invalidated=changed,
                             pid=self.process.pid)
            return kernel
        except BaseException:
            self.close()
            raise


def build(gart, source, output, env):
    """Use the production Go frontend and the same manifest freshness rule as gart run."""
    result = subprocess.run([str(gart), "build", "-timings", "-format=false", "-o", str(output), "./..."],
                            cwd=source, env=env, capture_output=True, text=True, timeout=120)
    if result.returncode:
        raise CompileError(result.stdout + result.stderr)
    config = output / ".dart_tool/package_config.json"
    manifests = [output / "pubspec.yaml", output / "goruntime/pubspec.yaml"]
    if not config.exists() or any(path.stat().st_mtime_ns > config.stat().st_mtime_ns for path in manifests):
        pub = subprocess.run(["dart", "pub", "get"], cwd=output, env=env, capture_output=True, text=True, timeout=120)
        if pub.returncode:
            raise CompileError(pub.stdout + pub.stderr)
    return result.stderr
