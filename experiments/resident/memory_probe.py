#!/usr/bin/env python3
"""Measure frontend memory across 100 edits with the HTTP playground fixture."""
import argparse
import http.server
import json
import os
from pathlib import Path
import subprocess
import threading
import time

from probe import Handler, ROOT, normalized
from session import Session, build


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    destination = args.output.resolve()
    if not destination.is_relative_to(ROOT / 'build'):
        parser.error('place output under build/')
    destination.mkdir(parents=True, exist_ok=False)
    env = dict(os.environ, GOMAXPROCS='4', GOFLAGS='-p=2')
    gart = destination / 'gart'
    subprocess.run(['go', 'build', '-o', str(gart), './cmd/gart'], cwd=ROOT, env=env, check=True)
    source = destination / 'source'
    source.mkdir()
    fixture = ROOT / 'experiments/resident/playground'
    (source / 'go.mod').write_bytes((fixture / 'go.mod').read_bytes())
    server = http.server.ThreadingHTTPServer(('127.0.0.1', 0), Handler)
    server.requests = 0
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    original = (fixture / 'main.go').read_text().replace('http://127.0.0.1:0/', f'http://127.0.0.1:{server.server_port}/')
    rows = []
    try:
        (source / 'main.go').write_text(original)
        native = subprocess.check_output(['go', 'run', '.'], cwd=source, env=env, stderr=subprocess.STDOUT, text=True)
        output = destination / 'output'
        with Session(output) as session:
            for index in range(101):
                name = f'{index:05d}'
                (source / 'main.go').write_text(original.replace('greet("world")', f'greet("{name}")'))
                build(gart, source, output, env)
                start = time.perf_counter()
                artifact = session.compile(output / 'bin/main.dart')
                elapsed = time.perf_counter() - start
                result = subprocess.check_output(['dart', 'run', str(artifact)], cwd=output, stderr=subprocess.STDOUT, text=True)
                if normalized(result) != normalized(native.replace('hello, world', 'hello, ' + name)):
                    raise RuntimeError(f'output mismatch at edit {index}')
                rss = int(subprocess.check_output(['ps', '-o', 'rss=', '-p', str(session.process.pid)], text=True))
                row = dict(edit=index, rss_kib=rss, compile_seconds=elapsed, mode=session.last['mode'])
                rows.append(row)
                print(json.dumps(row), flush=True)
            child = session.process
        if child.poll() is None:
            raise RuntimeError('compiler survived session close')
        if server.requests != 102:
            raise RuntimeError(f'HTTP requests: {server.requests}')
    finally:
        server.shutdown()
        server.server_close()
        thread.join()
    (destination / 'results.json').write_text(json.dumps(dict(rows=rows, http_requests=server.requests), indent=2)+'\n')


if __name__ == '__main__':
    main()
