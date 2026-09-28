import hashlib
import json
import os
from pathlib import Path
import re
import signal
import sys
import subprocess
import tempfile
import time
import unittest
import urllib.parse
import urllib.request

ROOT = Path(__file__).resolve().parents[1]


class ProductionContracts(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.workspace = ROOT / 'build/resident-production-contract'
        cls.workspace.mkdir(parents=True, exist_ok=True)
        cls.gart = cls.workspace / 'gart'
        subprocess.run(['go', 'build', '-o', str(cls.gart), './cmd/gart'], cwd=ROOT, check=True)

    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix='gart-contract-', dir='/tmp')
        self.root = Path(self.temp.name).resolve()
        self.env = dict(os.environ, TMPDIR=str(self.root), GOMAXPROCS='2', GOFLAGS='-p=2')
        self.namespace = str(Path('/tmp') / f'gart-{os.getuid()}' / hashlib.sha256(str(self.root).encode()).hexdigest()[:16])
        self.source = self.root / 'source'
        self.source.mkdir()
        (self.source / 'go.mod').write_text('module example.com/contract\n\ngo 1.27\n')
        self.output = self.root / 'output'
        self.argv = [str(self.gart), 'run', '-timings', '-format=false', '-o', str(self.output), '.']

    def processes(self):
        if sys.platform == 'linux':
            rows = []
            for directory in Path('/proc').glob('[0-9]*'):
                try:
                    cmd = (directory / 'cmdline').read_bytes().replace(b'\0', b' ').decode()
                    parent = re.search(r'^PPid:\s+(\d+)$', (directory / 'status').read_text(), re.MULTILINE)
                    if cmd and parent:
                        rows.append((int(directory.name), int(parent[1]), cmd))
                except (FileNotFoundError, ProcessLookupError):
                    pass
            return rows
        rows = subprocess.check_output(['ps', '-axo', 'pid,ppid,command'], text=True).splitlines()[1:]
        return [(int(parts[0]), int(parts[1]), parts[2]) for line in rows if len(parts := line.strip().split(None, 2)) == 3]

    def service(self):
        return next(pid for pid, _, cmd in self.processes() if self.namespace in cmd and '/service.dart ' in cmd)

    def tearDown(self):
        pids = [pid for pid, _, cmd in self.processes() if self.namespace in cmd and '/service.dart ' in cmd]
        for pid in pids:
            os.kill(pid, signal.SIGTERM)
        deadline = time.monotonic() + 5
        while any(pid in {p for p, _, _ in self.processes()} for pid in pids):
            self.assertLess(time.monotonic(), deadline, 'service did not terminate')
            time.sleep(.02)
        self.temp.cleanup()

    def run_gart(self):
        result = subprocess.run(self.argv, cwd=self.source, env=self.env, text=True, capture_output=True, timeout=120)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        return result

    def test_structural_go_edits(self):
        cases = [
            ('package main\nimport "fmt"\nfunc value() int{return 1}\nfunc main(){fmt.Println(value())}\n', '1\n', ''),
            ('package main\nimport "fmt"\nfunc value() string{return "two"}\nfunc main(){fmt.Println(value())}\n', 'two\n', ''),
            ('package main\nimport("fmt";"strings")\nfunc value() string{return "three"}\nfunc main(){fmt.Println(strings.ToUpper(value()))}\n', 'THREE\n', ''),
            ('package main\nfunc main(){println("four")}\n', '', 'four'),
        ]
        for index, (source, stdout, stderr) in enumerate(cases):
            (self.source / 'main.go').write_text(source)
            result = self.run_gart()
            self.assertEqual(result.stdout, stdout)
            if stderr:
                self.assertIn(stderr, result.stderr.splitlines())
            mode = 'cold' if index == 0 else 'incremental'
            self.assertIn('phase=kernel-' + mode + ' ', result.stderr)

    def test_concurrent_output_ownership(self):
        source = self.source / 'main.go'
        source.write_text('package main\nimport("fmt";"time")\nfunc main(){fmt.Println("ready");time.Sleep(2*time.Second);fmt.Println("done")}\n')
        log = self.root / 'owner.log'
        with log.open('w') as stream:
            first = subprocess.Popen(self.argv, cwd=self.source, env=self.env, stdout=stream,
                                     stderr=stream, start_new_session=True)
            try:
                deadline = time.monotonic() + 30
                while 'ready\n' not in log.read_text():
                    self.assertIsNone(first.poll(), log.read_text())
                    self.assertLess(time.monotonic(), deadline)
                    time.sleep(.02)
                conflict = subprocess.run(self.argv, cwd=self.source, env=self.env, capture_output=True, text=True, timeout=10)
                self.assertEqual(conflict.returncode, 1)
                self.assertIn('output directory is in use', conflict.stderr)
                self.assertEqual(first.wait(timeout=10), 0, log.read_text())
            finally:
                if first.poll() is None:
                    os.killpg(first.pid, signal.SIGKILL)
                    first.wait()
        other = list(self.argv)
        other[other.index('-o') + 1] = str(self.root / 'other-output')
        result = subprocess.run(other, cwd=self.source, env=self.env, capture_output=True, text=True, timeout=120)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        service = self.service()
        source.write_text(source.read_text().replace('"done"', '"next"'))
        processes = [subprocess.Popen(argv, cwd=self.source, env=self.env, stdout=subprocess.PIPE,
                                      stderr=subprocess.PIPE, text=True, start_new_session=True)
                     for argv in [self.argv, other]]
        try:
            for process in processes:
                stdout, stderr = process.communicate(timeout=120)
                self.assertEqual(process.returncode, 0, stdout + stderr)
                self.assertIn('next\n', stdout)
                self.assertIn('phase=kernel-incremental ', stderr)
            self.assertEqual(self.service(), service)
        finally:
            for process in processes:
                if process.poll() is None:
                    os.killpg(process.pid, signal.SIGKILL)
                    process.wait()

    def test_worker_and_service_crash_recovery(self):
        source = self.source / 'main.go'
        source.write_text('package main\nfunc main(){println("first")}\n')
        result = self.run_gart()
        self.assertEqual(result.stdout, '')
        self.assertIn('first', result.stderr.splitlines())
        service = self.service()
        worker = next(pid for pid, parent, cmd in self.processes() if parent == service and 'frontend_server_aot' in cmd)
        os.kill(worker, signal.SIGKILL)
        time.sleep(.1)
        source.write_text(source.read_text().replace('first', 'other'))
        result = self.run_gart()
        self.assertEqual(result.stdout, '')
        self.assertIn('other', result.stderr.splitlines())
        self.assertIn('phase=kernel-cold ', result.stderr)
        child = next(pid for pid, parent, cmd in self.processes() if parent == service and 'frontend_server_aot' in cmd)
        os.kill(service, signal.SIGKILL)
        deadline = time.monotonic() + 5
        while child in {pid for pid, _, _ in self.processes()}:
            self.assertLess(time.monotonic(), deadline, 'compiler survived service death')
            time.sleep(.02)
        source.write_text(source.read_text().replace('other', 'third'))
        result = self.run_gart()
        self.assertEqual(result.stdout, '')
        self.assertIn('third', result.stderr.splitlines())
        self.assertIn('phase=kernel-cold ', result.stderr)
        self.assertNotEqual(self.service(), service)

    def test_incremental_debugger_and_cli_signals(self):
        source = self.source / 'main.go'
        source.write_text('package main\nimport("fmt";"time")\nfunc main(){fmt.Println("ready");time.Sleep(2*time.Second);fmt.Println("done")}\n')
        self.run_gart()
        source.write_text(source.read_text().replace('"done"', '"next"'))
        result = self.run_gart()
        self.assertIn('phase=kernel-incremental ', result.stderr)
        artifact, = (self.output / '.gart/kernel').glob('*.dill')
        part = self.output / 'bin/main/main.part.dart'
        line = [i for i, text in enumerate(part.read_text().splitlines(), 1) if '$fmt.Println' in text][1]
        for entry in [self.output / 'bin/main.dart', artifact]:
            log = self.root / 'debug.log'
            with log.open('w') as stream:
                process = subprocess.Popen(['dart', 'run', '--observe=0', '--no-pause-isolates-on-exit', str(entry)],
                                           cwd=self.output, env=self.env, stdout=stream, stderr=stream, start_new_session=True)
                try:
                    deadline = time.monotonic() + 30
                    while True:
                        text = log.read_text()
                        found = re.search(r'The Dart VM service is listening on (http://\S+/)', text)
                        if found and 'ready\n' in text:
                            base = found[1]
                            break
                        self.assertIsNone(process.poll(), text)
                        self.assertLess(time.monotonic(), deadline, text)
                        time.sleep(.02)

                    def rpc(method, **params):
                        with urllib.request.urlopen(base + method + '?' + urllib.parse.urlencode(params), timeout=5) as response:
                            result = json.load(response)
                        self.assertNotIn('error', result)
                        return result['result']

                    isolate = rpc('getVM')['isolates'][0]['id']
                    scripts = rpc('getScripts', isolateId=isolate)['scripts']
                    uri = next(s['uri'] for s in scripts if s['uri'].endswith('/main/main.part.dart'))
                    self.assertTrue(rpc('addBreakpointWithScriptUri', isolateId=isolate, scriptUri=uri, line=line)['resolved'])
                    deadline = time.monotonic() + 10
                    while True:
                        state = rpc('getIsolate', isolateId=isolate)
                        if state['pauseEvent']['kind'] == 'PauseBreakpoint':
                            break
                        self.assertLess(time.monotonic(), deadline)
                        time.sleep(.02)
                    rpc('resume', isolateId=isolate)
                    self.assertEqual(process.wait(timeout=10), 0, log.read_text())
                    self.assertIn('next\n', log.read_text())
                finally:
                    if process.poll() is None:
                        os.killpg(process.pid, signal.SIGKILL)
                        process.wait()
        service = self.service()
        for sig in [signal.SIGINT, signal.SIGTERM]:
            log = self.root / 'signal.log'
            with log.open('w') as stream:
                process = subprocess.Popen(self.argv, cwd=self.source, env=self.env, stdout=stream, stderr=stream, start_new_session=True)
                try:
                    deadline = time.monotonic() + 30
                    while 'ready\n' not in log.read_text():
                        self.assertIsNone(process.poll(), log.read_text())
                        self.assertLess(time.monotonic(), deadline)
                        time.sleep(.02)
                    os.killpg(process.pid, sig)
                    self.assertEqual(process.wait(timeout=10), -sig)
                finally:
                    if process.poll() is None:
                        os.killpg(process.pid, signal.SIGKILL)
                        process.wait()
            self.assertEqual(self.service(), service)
        source.write_text(source.read_text().replace('"next"', '"last"'))
        self.assertIn('phase=kernel-incremental ', self.run_gart().stderr)


if __name__ == '__main__':
    unittest.main()
