#!/usr/bin/env python3
"""Measure imported Go algorithms with isolated, content-addressed VM kernels."""
import hashlib
import json
import os
from pathlib import Path
import shutil
import statistics
import time
from urllib.parse import unquote, urljoin, urlparse

from probe import HERE, OUT, REVISION, ROOT, prepare, run

WORK = OUT / 'imported'


def digest(value):
    return hashlib.sha256(json.dumps(value, sort_keys=True, separators=(',', ':')).encode()).hexdigest()


def file_hash(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def files_identity(paths):
    return {str(p): file_hash(p) for p in sorted(set(paths)) if p.is_file()}


def objects(text):
    decoder = json.JSONDecoder()
    while text.strip():
        item, end = decoder.raw_decode(text.lstrip())
        yield item
        text = text.lstrip()[end:]


def file_uri(uri):
    parsed = urlparse(uri)
    if parsed.scheme != 'file' or parsed.netloc:
        raise ValueError(f'Expected a local package URI: {uri}')
    return Path(unquote(parsed.path))


class Cache:
    def __init__(self, tools, module, runtime_config, directory):
        self.tools = tools
        self.module = module
        self.config = runtime_config
        self.directory = directory
        self.defines = []

    def identity(self):
        phases = {}
        phase = time.monotonic()
        environment = json.loads(run(['go', 'env', '-json', 'GOVERSION', 'GOOS', 'GOARCH', 'GOAMD64',
                                      'GOARM64', 'GOEXPERIMENT', 'CGO_ENABLED', 'GOFLAGS', 'GOWORK',
                                      'GOROOT', 'GOPATH', 'GOMOD', 'GOMODCACHE'], cwd=self.module)[0].stdout)
        phases['go_environment'] = time.monotonic() - phase
        phase = time.monotonic()
        graph = list(objects(run(['go', 'list', '-deps', '-json', '.'], cwd=self.module)[0].stdout))
        phases['go_discovery'] = time.monotonic() - phase
        phase = time.monotonic()
        paths = [self.module / 'go.mod', self.module / 'go.sum']
        packages = []
        for p in graph:
            sources = [Path(p['Dir']) / name for name in p.get('GoFiles', [])]
            paths.extend(sources)
            module = p.get('Module')
            if module:
                metadata = module.get('Replace', module)
                if metadata.get('GoMod'):
                    paths.append(Path(metadata['GoMod']))
                    paths.append(Path(metadata['GoMod']).with_name('go.sum'))
            packages.append(dict(path=p['ImportPath'], imports=p.get('Imports', []), sources=list(map(str, sources)), module=module))
        go_files = files_identity(paths)
        phases['go_hash'] = time.monotonic() - phase
        phase = time.monotonic()
        runtime_paths = [self.config, HERE / 'runtime.dart']
        for package in json.loads(self.config.read_text())['packages']:
            root_uri = urljoin(self.config.as_uri(), package['rootUri'])
            root = file_uri(root_uri)
            # Package roots denote directories even without a trailing slash.
            library = file_uri(urljoin(root_uri.rstrip('/') + '/', package.get('packageUri', '')))
            runtime_paths.extend(library.rglob('*.dart'))
            runtime_paths.extend([root / 'pubspec.yaml', root / 'pubspec.lock'])
        sdk = self.tools['sdk']
        revision = (sdk / 'revision').read_text().strip()
        if revision != REVISION:
            raise ValueError('Installed SDK revision differs from the prototype pin')
        runtime = dict(sdk=str(sdk), revision=revision, target='vm', defines=self.defines,
                       files=files_identity(runtime_paths))
        phases['runtime_hash'] = time.monotonic() - phase
        phase = time.monotonic()
        application = dict(runtime=digest(runtime), environment=environment, packages=packages,
                           files=go_files, directory=str(self.directory),
                           frontend=file_hash(self.tools['frontend']), emitter=file_hash(self.tools['emitter']))
        phases['tool_hash'] = time.monotonic() - phase
        return digest(runtime), application, phases

    def artifact(self, stem):
        kernel = stem.with_suffix('.dill')
        receipt = stem.with_suffix('.json')
        if not kernel.is_file() or not receipt.is_file():
            return None
        try:
            metadata = json.loads(receipt.read_text())
        except (ValueError, OSError):
            return None
        if (not isinstance(metadata, dict) or not isinstance(metadata.get('functions'), list)
                or not all(isinstance(name, str) for name in metadata['functions'])
                or metadata.get('sha256') != file_hash(kernel)):
            return None
        return kernel, metadata

    def save(self, stem, temporary, metadata):
        kernel = stem.with_suffix('.dill')
        temporary.replace(kernel)
        receipt = stem.with_suffix('.json')
        pending = stem.with_suffix('.json.tmp')
        pending.write_text(json.dumps(dict(metadata, sha256=file_hash(kernel)), indent=2) + '\n')
        pending.replace(receipt)
        return kernel

    def build(self, backend):
        start = time.monotonic()
        runtime_key, application, fingerprint_phases = self.identity()
        phase = dict(fingerprint=time.monotonic() - start, runtime_compile=0.0, frontend=0.0, kernel=0.0)
        key = digest(dict(application, backend=backend))
        self.directory.mkdir(parents=True, exist_ok=True)
        stem = self.directory / f'{backend}-{key}'
        cached = self.artifact(stem)
        runtime_hit = None
        frontend_phases = None
        if cached:
            kernel, metadata = cached
            functions = metadata['functions']
        else:
            dart = self.tools['dart']
            flags = ['--packages=' + str(self.config), *self.defines]
            if backend == 'direct':
                runtime_stem = self.directory / f'runtime-{runtime_key}'
                cached_runtime = self.artifact(runtime_stem)
                runtime_hit = cached_runtime is not None
                if cached_runtime:
                    runtime = cached_runtime[0]
                else:
                    temporary = runtime_stem.with_suffix('.tmp.dill')
                    _, phase['runtime_compile'] = run([dart, 'compile', 'kernel', *flags, '-o', temporary, HERE / 'runtime.dart'])
                    runtime = self.save(runtime_stem, temporary, dict(functions=[]))
            output = self.directory / f'{backend}-work'
            result, phase['frontend'] = run([self.tools['frontend'], self.module, output, backend])
            frontend = json.loads(result.stdout)
            functions, frontend_phases = frontend['functions'], frontend['phases']
            temporary = stem.with_suffix('.tmp.dill')
            if backend == 'direct':
                _, phase['kernel'] = run([self.tools['emitter'], runtime, output / 'program.json', temporary])
            else:
                _, phase['kernel'] = run([dart, 'compile', 'kernel', *flags, '-o', temporary, output / 'main.dart'])
            kernel = self.save(stem, temporary, dict(functions=functions))
        phase['total'] = time.monotonic() - start
        result, execution = run([self.tools['dart'], 'run', kernel])
        return dict(backend=backend, key=key, runtime_key=runtime_key, hit=cached is not None,
                    runtime_hit=runtime_hit, phases=phase, fingerprint_phases=fingerprint_phases, frontend_phases=frontend_phases,
                    execute_seconds=execution, functions=functions, kernel=str(kernel),
                    stdout=result.stdout, stderr=result.stderr)


def compact(row):
    return {k: v for k, v in row.items() if k not in ('stdout', 'stderr')}


def main():
    os.environ['GOWORK'] = 'off'
    tools = prepare()
    WORK.mkdir(parents=True, exist_ok=True)
    module = WORK / 'module'
    runtime = WORK / 'goruntime'
    # Mutations apply only to these disposable copies.
    dependency = WORK / 'http2'
    for path in [module, runtime, dependency, WORK / 'cache', WORK / 'timed-cache']:
        if path.exists():
            shutil.rmtree(path)
    shutil.copytree(HERE / 'fixtures/imported', module)
    shutil.copytree(ROOT / 'runtime/goruntime/lib', runtime / 'lib')
    for name in ['pubspec.yaml', 'pubspec.lock']:
        source = ROOT / 'runtime/goruntime' / name
        if source.exists():
            shutil.copy2(source, runtime / name)
    config = WORK / 'package_config.json'
    packages = json.loads(tools['runtime_config'].read_text())
    for package in packages['packages']:
        if package['name'] == 'http2':
            root = file_uri(urljoin(tools['runtime_config'].as_uri(), package['rootUri']))
            shutil.copytree(root / 'lib', dependency / 'lib')
            shutil.copy2(root / 'pubspec.yaml', dependency / 'pubspec.yaml')
            package['rootUri'] = dependency.as_uri()
        package['rootUri'] = (runtime.as_uri() + '/') if package['name'] == 'goruntime' else urljoin(tools['runtime_config'].as_uri(), package['rootUri'])
    config.write_text(json.dumps(packages, indent=2) + '\n')
    print_file = runtime / 'lib/src/print.dart'
    original_print = print_file.read_text()
    needle = "$platformStderrText('${parts.join(' ')}\\n');"
    instrumented = original_print.replace(needle, "$platformStderrText('${const String.fromEnvironment('kernel.probe.prefix', defaultValue: '')}${parts.join(' ')}\\n');")
    assert instrumented != original_print
    print_file.write_text(instrumented)
    cache = Cache(tools, module, config, WORK / 'cache')
    checks = []

    def compare(label, *, hit, runtime_hit=None, prefix='', order=('direct', 'source')):
        native = run(['go', 'run', '.'], cwd=module)[0]
        expected = ''.join(prefix + line for line in native.stderr.splitlines(keepends=True))
        results = [cache.build(backend) for backend in order]
        for row in results:
            assert row['hit'] == hit, (label, compact(row))
            assert row['stdout'] == native.stdout == '' and row['stderr'] == expected, (label, row['backend'])
            if row['backend'] == 'direct' and not hit and runtime_hit is not None:
                assert row['runtime_hit'] == runtime_hit, (label, compact(row))
        assert results[0]['functions'] == results[1]['functions']
        checks.append(dict(name=label, output_lines=len(expected.splitlines()), results=list(map(compact, results))))
        print('PASS', label, flush=True)
        return results

    baseline = compare('cold-artifacts', hit=False, runtime_hit=False)
    compare('unchanged', hit=True)
    main_file = module / 'main.go'
    main_text = main_file.read_text()
    old_stat = main_file.stat()
    main_file.write_text(main_text.replace('uint64(17)', 'uint64(19)'))
    os.utime(main_file, ns=(old_stat.st_atime_ns, old_stat.st_mtime_ns))
    compare('application-same-size-and-mtime-edit', hit=False, runtime_hit=True)
    compare('application-edit-repeat', hit=True)
    settings = module / 'settings/settings.go'
    settings_text = settings.read_text()
    settings.write_text(settings_text.replace('Bias = 7', 'Bias = 9'))
    compare('transitive-import-edit', hit=False, runtime_hit=True)
    added = settings.with_name('extra.go')
    added.write_text('package settings\nfunc Extra() int { return 11 }\n')
    compare('added-dependency-file-only', hit=False, runtime_hit=True)
    added.unlink()
    compare('removed-file-restores-known-graph', hit=True)
    module_file = module / 'go.mod'
    module_text = module_file.read_text()
    module_file.write_text(module_text + '\n// Cache identity probe.\n')
    compare('module-metadata-edit', hit=False, runtime_hit=True)
    config_text = config.read_text()
    changed_config = json.loads(config_text)
    changed_config['generator'] = 'kernel-cache-probe'
    config.write_text(json.dumps(changed_config, indent=2) + '\n')
    compare('package-config-edit', hit=False, runtime_hit=False)
    print_file.write_text(instrumented.replace("defaultValue: ''", "defaultValue: 'runtime:'"))
    compare('runtime-source-edit', hit=False, runtime_hit=False, prefix='runtime:')
    dependency_file = dependency / 'lib/transport.dart'
    dependency_text = dependency_file.read_text()
    dependency_file.write_text(dependency_text + '\n// Cache dependency probe.\n')
    compare('transitive-dart-dependency-edit', hit=False, runtime_hit=False, prefix='runtime:')
    cache.defines = ['-Dkernel.probe.prefix=option:']
    compare('compiler-option-edit', hit=False, runtime_hit=False, prefix='option:')
    compare('compiler-option-repeat', hit=True, prefix='option:')
    cache.defines = []
    main_file.write_text(main_text)
    settings.write_text(settings_text)
    module_file.write_text(module_text)
    config.write_text(config_text)
    print_file.write_text(instrumented)
    dependency_file.write_text(dependency_text)
    compare('restored-inputs', hit=True)
    for row in baseline:
        Path(row['kernel']).write_bytes(b'corrupt')
    compare('corrupt-final-artifacts', hit=False, runtime_hit=True)
    for row in baseline:
        Path(row['kernel']).unlink()
    seed = cache.directory / ('runtime-' + baseline[0]['runtime_key'] + '.dill')
    seed.write_bytes(b'corrupt')
    compare('missing-final-and-corrupt-runtime', hit=False, runtime_hit=False)
    for row in baseline:
        Path(row['kernel']).with_suffix('.json').write_text('{invalid')
    compare('corrupt-receipt', hit=False, runtime_hit=True)
    for row in baseline:
        Path(row['kernel']).with_suffix('.json').write_text(json.dumps(dict(sha256=file_hash(Path(row['kernel'])))))
    compare('incomplete-receipt', hit=False, runtime_hit=True)

    original_tools = cache.tools
    fake_sdk = WORK / 'wrong-sdk'
    fake_sdk.mkdir(exist_ok=True)
    (fake_sdk / 'revision').write_text('0' * 40)
    cache.tools = dict(tools, sdk=fake_sdk)
    try:
        cache.build('direct')
    except ValueError as error:
        assert 'SDK revision' in str(error)
    else:
        raise AssertionError('Mismatched SDK accepted a cached artifact')
    cache.tools = original_tools
    print('PASS cache SDK rejection', flush=True)

    direct = next(r for r in baseline if r['backend'] == 'direct')
    executable = WORK / 'imported.exe'
    run([tools['dart'], 'compile', 'exe', '-o', executable, direct['kernel']])
    aot = run([executable])[0]
    assert aot.stdout == direct['stdout'] and aot.stderr == direct['stderr']
    print('PASS imported AOT', flush=True)

    cache.directory = WORK / 'timed-cache'
    trials = []
    for trial in range(5):
        if cache.directory.exists():
            shutil.rmtree(cache.directory)
        main_file.write_text(main_text)
        settings.write_text(settings_text)
        order = ('direct', 'source') if trial % 2 == 0 else ('source', 'direct')
        for scenario in ['cold-artifacts', 'unchanged', 'application-edit', 'import-edit']:
            if scenario == 'application-edit':
                main_file.write_text(main_text.replace('uint64(17)', 'uint64(19)'))
            if scenario == 'import-edit':
                settings.write_text(settings_text.replace('Bias = 7', 'Bias = 9'))
            native = run(['go', 'run', '.'], cwd=module)[0]
            rows = [cache.build(backend) for backend in order]
            for row in rows:
                assert row['hit'] == (scenario == 'unchanged')
                assert (row['stdout'], row['stderr']) == (native.stdout, native.stderr)
                trials.append(dict(trial=trial, scenario=scenario, **compact(row)))
            assert rows[0]['functions'] == rows[1]['functions']
        print('PASS timing trial', trial + 1, flush=True)
    medians = {}
    for scenario in ['cold-artifacts', 'unchanged', 'application-edit', 'import-edit']:
        medians[scenario] = {}
        for backend in ['direct', 'source']:
            rows = [r for r in trials if r['scenario'] == scenario and r['backend'] == backend]
            medians[scenario][backend] = {phase: statistics.median(r['phases'][phase] for r in rows) for phase in rows[0]['phases']}
            medians[scenario][backend]['execute'] = statistics.median(r['execute_seconds'] for r in rows)
            medians[scenario][backend]['build_and_execute'] = statistics.median(r['phases']['total'] + r['execute_seconds'] for r in rows)
    source_identity = dict(go=run(['go', 'version'])[0].stdout.strip(),
                           files=files_identity(Path(run(['go', 'env', 'GOROOT'])[0].stdout.strip()).joinpath('src/math/bits').glob('*.go')))
    report = dict(source_identity=source_identity, dart=tools['version'], revision=REVISION, setup=tools['setup'], checks=checks,
                  aot=True, sdk_rejected=True, medians=medians, trials=trials)
    (WORK / 'results.json').write_text(json.dumps(report, indent=2) + '\n')
    print(json.dumps(medians, indent=2), flush=True)
    print('All imported-package probes pass.', flush=True)


if __name__ == '__main__':
    main()
