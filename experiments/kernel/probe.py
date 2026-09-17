#!/usr/bin/env python3
"""Build and measure the isolated Dart 3.13.3 kernel prototype."""
import json
from pathlib import Path
import shutil
import statistics
import subprocess
import time
from urllib.parse import urljoin

ROOT = Path(__file__).resolve().parents[2]
HERE = Path(__file__).resolve().parent
OUT = ROOT / 'build/kernel-prototype'
REVISION = '1d1a730ef918d602aedafc939a4cf5940e7589ab'
SDK_VERSION = '3.13.3'


def run(command, *, cwd=ROOT, expected=0):
    start = time.monotonic()
    result = subprocess.run([str(x) for x in command], cwd=cwd, text=True, capture_output=True, timeout=120)
    elapsed = time.monotonic() - start
    if result.returncode != expected:
        raise RuntimeError(f'{command}: exit {result.returncode}\n{result.stdout}\n{result.stderr}')
    return result, elapsed


def prepare():
    OUT.mkdir(parents=True, exist_ok=True)
    dart = Path(shutil.which('dart')).resolve()
    sdk = dart.parent.parent
    version = run([dart, '--version'])[0].stdout.strip()
    if not version.startswith(f'Dart SDK version: {SDK_VERSION} ') or (sdk / 'revision').read_text().strip() != REVISION:
        raise SystemExit(f'This prototype requires Dart {SDK_VERSION}, revision {REVISION}.')
    checkout = OUT / 'sdk'
    if not checkout.exists():
        run(['git', 'clone', '--filter=blob:none', '--depth', '1', '--branch', SDK_VERSION, '--sparse',
             'https://github.com/dart-lang/sdk.git', checkout])
    actual = run(['git', 'rev-parse', 'HEAD'], cwd=checkout)[0].stdout.strip()
    if actual != REVISION:
        raise SystemExit(f'SDK source revision differs: {actual}')
    run(['git', 'sparse-checkout', 'set', 'pkg/kernel', 'pkg/_fe_analyzer_shared', 'pkg/meta'], cwd=checkout)
    run(['git', 'diff', '--exit-code', 'HEAD', '--', 'pkg/kernel', 'pkg/_fe_analyzer_shared', 'pkg/meta'], cwd=checkout)
    runtime_config = ROOT / 'runtime/goruntime/.dart_tool/package_config.json'
    run([dart, 'pub', 'get'], cwd=ROOT / 'runtime/goruntime')
    pinned = ['kernel', '_fe_analyzer_shared', 'meta']
    packages = dict(configVersion=2, packages=[dict(name=name, rootUri=(checkout / 'pkg' / name).as_uri() + '/',
        packageUri='lib/', languageVersion='3.13') for name in pinned])
    for package in json.loads(runtime_config.read_text())['packages']:
        if package['name'] not in pinned:
            packages['packages'].append(package | {'rootUri': urljoin(runtime_config.as_uri(), package['rootUri'])})
    config = HERE / '.dart_tool/package_config.json'
    config.parent.mkdir(exist_ok=True)
    config.write_text(json.dumps(packages, indent=2) + '\n')
    frontend = OUT / 'frontend-bin'
    emitter = OUT / 'emit-bin'
    setup = {}
    _, setup['go_frontend'] = run(['go', 'build', '-o', frontend, HERE / 'frontend/main.go'])
    _, setup['kernel_emitter'] = run([dart, 'compile', 'exe', '-Dsdk_hash=' + REVISION[:10],
                                     '--packages=' + str(config), '-o', emitter, HERE / 'emit.dart'])
    return dict(dart=dart, sdk=sdk, version=version, frontend=frontend, emitter=emitter,
                runtime_config=runtime_config, setup=setup)


def main():
    tools = prepare()
    dart, sdk = tools['dart'], tools['sdk']
    frontend, emitter = tools['frontend'], tools['emitter']
    runtime_config, setup, version = tools['runtime_config'], tools['setup'], tools['version']
    wasm = [sdk / 'bin/dartaotruntime', sdk / 'bin/snapshots/dart2wasm_product.snapshot',
            '--libraries-spec=' + str(sdk / 'lib/libraries.json')]
    seeds = {target: OUT / f'seed-{target}.dill' for target in ['vm', 'js', 'wasm']}
    _, setup['runtime_vm'] = run([dart, 'compile', 'kernel', '--packages=' + str(runtime_config),
                                  '-o', seeds['vm'], HERE / 'runtime.dart'])
    _, setup['runtime_js'] = run([dart, 'compile', 'js', '--cfe-only', '--packages=' + str(runtime_config),
                                  '-o', seeds['js'], HERE / 'runtime.dart'])
    _, setup['runtime_wasm'] = run(wasm + ['--phases=cfe', '--packages=' + str(runtime_config),
                                         HERE / 'runtime.dart', seeds['wasm']])
    print('Setup complete', json.dumps(setup), flush=True)
    checks = []
    for name in ['sum', 'recursion', 'integers']:
        fixture = HERE / f'fixtures/{name}.go'
        directory = OUT / f'fixture-{name}'
        run([frontend, fixture, directory, 'both'])
        native = directory / 'native'
        run(['go', 'build', '-o', native, fixture])
        expected = run([native])[0]
        run([dart, 'compile', 'kernel', '--packages=' + str(runtime_config), '-o', directory / 'source.dill', directory / 'main.dart'])
        source = run([dart, 'run', directory / 'source.dill'])[0]
        assert (source.stdout, source.stderr) == (expected.stdout, expected.stderr), name
        run([emitter, seeds['vm'], directory / 'program.json', directory / 'direct.dill'])
        assert (directory / 'direct.dill').read_bytes()[8:18] == REVISION[:10].encode()
        direct = run([dart, 'run', directory / 'direct.dill'])[0]
        assert (direct.stdout, direct.stderr) == (expected.stdout, expected.stderr), name
        run([dart, 'compile', 'exe', '-o', directory / 'direct.exe', directory / 'direct.dill'])
        aot = run([directory / 'direct.exe'])[0]
        assert (aot.stdout, aot.stderr) == (expected.stdout, expected.stderr), name
        # The prototype's int nodes do not implement GoInt on JavaScript.
        if name != 'integers':
            for target in ['js', 'wasm']:
                kernel = directory / f'direct-{target}.dill'
                run([emitter, seeds[target], directory / 'program.json', kernel])
                if target == 'js':
                    run([dart, 'compile', 'js', '-o', directory / 'direct.js', kernel])
                    web = run(['node', directory / 'direct.js'])[0]
                else:
                    run(wasm + ['--phases=tfa,codegen', kernel, directory / 'direct.wasm'])
                    runner = directory / 'run.mjs'
                    runner.write_text("import {readFile} from 'node:fs/promises';\nimport {compile} from './direct.mjs';\n"
                        "const app = await (await compile(await readFile(new URL('./direct.wasm', import.meta.url)))).instantiate();\napp.invokeMain();\n")
                    web = run(['node', runner])[0]
                assert web.stdout == expected.stderr and web.stderr == '', (name, target)
        checks.append(dict(fixture=name, native=True, source=True, vm=True, aot=True,
                           js=name != 'integers', wasm=name != 'integers', output=expected.stderr))
        print('PASS', name, flush=True)
    unsupported = OUT / 'unsupported.go'
    unsupported.write_text('package main\nfunc main() { value := 1.5; println(value) }\n')
    result, _ = run([frontend, unsupported, OUT / 'unsupported', 'both'], expected=1)
    assert 'unsupported type' in result.stderr
    bad = bytearray((OUT / 'fixture-sum/direct.dill').read_bytes())
    bad[8:18] = b'ffffffffff'
    bad_path = OUT / 'wrong-sdk.dill'
    bad_path.write_bytes(bad)
    rejected = subprocess.run([str(dart), 'run', str(bad_path)], text=True, capture_output=True, timeout=10)
    assert rejected.returncode != 0 and 'SDK hash' in rejected.stderr, rejected
    # The kernel path must not consume the application Dart files.
    directory = OUT / 'fixture-sum'
    for path in directory.rglob('*.dart'):
        path.unlink()
    run([emitter, seeds['vm'], directory / 'program.json', directory / 'no-source.dill'])
    assert run([dart, 'run', directory / 'no-source.dill'])[0].stderr == 'sum 30\n'
    measurements = []
    for count in [10, 250, 2500]:
        fixture = OUT / f'scale-{count}.go'
        functions = ''.join(f'func f{i}(x int) int {{ return x*3+{i} }}\n' for i in range(count))
        calls = ''.join(f' total += f{i}(value)\n' for i in range(count))
        fixture.write_text('package main\n' + functions + 'func main() { value:=7; total:=0\n' + calls + 'println(total)\n}\n')
        directory = OUT / f'scale-{count}'
        _, frontend_seconds = run([frontend, fixture, directory, 'both'])
        samples = []
        for trial in range(5):
            commands = {
                'direct': [emitter, seeds['vm'], directory / 'program.json', directory / 'direct.dill'],
                'source': [dart, 'compile', 'kernel', '--packages=' + str(runtime_config), '-o', directory / 'source.dill', directory / 'main.dart'],
            }
            order = ['direct', 'source'] if trial % 2 == 0 else ['source', 'direct']
            for mode in order:
                _, seconds = run(commands[mode])
                samples.append(dict(trial=trial, mode=mode, seconds=seconds))
        direct = run([dart, 'run', directory / 'direct.dill'])[0]
        source = run([dart, 'run', directory / 'source.dill'])[0]
        assert direct.stderr == source.stderr == str(sum(21+i for i in range(count))) + '\n'
        assert direct.stdout == source.stdout == ''
        medians = {mode: statistics.median(s['seconds'] for s in samples if s['mode'] == mode) for mode in ['direct', 'source']}
        row = dict(functions=count, frontend_seconds=frontend_seconds, medians=medians, samples=samples)
        measurements.append(row)
        print('MEDIANS', count, json.dumps(medians), flush=True)
    report = dict(dart=version, revision=REVISION, setup=setup, checks=checks,
                  unsupported_rejected=True, wrong_sdk_rejected=True, no_application_source=True, measurements=measurements)
    (OUT / 'results.json').write_text(json.dumps(report, indent=2) + '\n')
    print('All prototype probes pass.', flush=True)


if __name__ == '__main__':
    main()
