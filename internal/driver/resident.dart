import 'dart:async';
import 'dart:convert';
import 'dart:io';

const idleLimit = Duration(seconds: 60);
const retainedRssKiB = 1024 * 1024;
const editLimit = 100;

class Compiler {
  final Process process;
  final Directory scratch;
  final StreamIterator<String> lines;
  final String identity;
  final String entry;
  final Map<String, String> dependencies = {};
  Map<String, dynamic> sources = {};
  int requests = 0;
  bool exited = false;
  final idle = Stopwatch()..start();

  Compiler(this.process, this.scratch, this.identity, this.entry)
    : lines = StreamIterator(
        process.stdout.transform(utf8.decoder).transform(const LineSplitter()),
      ) {
    process.stderr.drain<void>();
    process.exitCode.then((_) {
      exited = true;
    });
  }

  Future<void> close() async {
    process.kill();
    try {
      await process.exitCode.timeout(const Duration(seconds: 3));
    } on TimeoutException {
      process.kill(ProcessSignal.sigkill);
      await process.exitCode;
    }
    await lines.cancel();
    if (scratch.existsSync()) scratch.deleteSync(recursive: true);
  }

  Future<String> line() async {
    if (!await lines.moveNext())
      throw StateError('resident compiler exited before a complete response');
    return lines.current;
  }

  Future<String> compile(Map<String, dynamic> next) async {
    if (requests == 0) {
      process.stdin.writeln('compile ${File(entry).uri}');
    } else {
      final boundary = 'gart-${DateTime.now().microsecondsSinceEpoch}';
      process.stdin.writeln('recompile $boundary');
      final changed = {
        ...sources.keys,
        ...next.keys,
      }.where((p) => sources[p] != next[p]).toList()..sort();
      final invalidated = changed
          .map((path) => File(path).uri.toString())
          .toSet();
      for (final dependency in dependencies.entries) {
        if (changed.contains(dependency.value)) invalidated.add(dependency.key);
      }
      for (final uri in invalidated) {
        process.stdin.writeln(uri);
      }
      process.stdin.writeln(boundary);
    }
    await process.stdin.flush();
    final diagnostics = <String>[];
    var value = await line();
    while (!value.startsWith('result ')) {
      diagnostics.add(value);
      value = await line();
    }
    final boundary = value.substring(7);
    value = await line();
    while (value != boundary) {
      diagnostics.add(value);
      value = await line();
    }
    value = await line();
    while (!value.startsWith('$boundary ')) {
      if (!value.startsWith('+file:') && !value.startsWith('-file:')) {
        throw StateError('unsupported compiler dependency: $value');
      }
      final uri = value.substring(1);
      if (value.startsWith('+')) {
        dependencies[uri] = File.fromUri(Uri.parse(uri))
            .resolveSymbolicLinksSync();
      } else {
        dependencies.remove(uri);
      }
      value = await line();
    }
    final split = value.lastIndexOf(' ');
    if (int.parse(value.substring(split + 1)) != 0) {
      throw StateError(diagnostics.join('\n'));
    }
    final result = value.substring(boundary.length + 1, split);
    if (!File(result).absolute.path.startsWith('${scratch.path}/')) {
      throw StateError('unexpected compiler artifact path');
    }
    process.stdin.writeln('accept\nreset');
    await process.stdin.flush();
    sources = next;
    requests++;
    idle.reset();
    return result;
  }
}

Future<int?> memoryKiB(int pid) async {
  try {
    if (Platform.isLinux) {
      final status = await File('/proc/$pid/status').readAsString();
      final match = RegExp(
        r'^VmRSS:\s+(\d+)\s+kB$',
        multiLine: true,
      ).firstMatch(status);
      return match == null ? null : int.parse(match[1]!);
    }
    final result = await Process.run('ps', ['-o', 'rss=', '-p', '$pid']);
    return result.exitCode == 0
        ? int.tryParse(result.stdout.toString().trim())
        : null;
  } on FileSystemException {
    return null;
  } on ProcessException {
    return null;
  }
}

Future<void> main(List<String> args) async {
  final socketPath = args[0];
  final sdk = args[1];
  final directory = Directory(File(socketPath).parent.path);
  final server = await ServerSocket.bind(
    InternetAddress(socketPath, type: InternetAddressType.unix),
    0,
  );
  final compilers = <String, Compiler>{};
  var busy = false;
  final idle = Stopwatch()..start();
  var closing = false;
  Future<void> close() async {
    if (closing) return;
    closing = true;
    final socketFile = File(socketPath);
    if (FileSystemEntity.typeSync(socketPath, followLinks: false) !=
        FileSystemEntityType.notFound)
      socketFile.deleteSync();
    await server.close();
    for (final compiler in compilers.values.toList()) {
      await compiler.close();
    }
    compilers.clear();
  }

  final signals = [ProcessSignal.sigterm, ProcessSignal.sigint]
      .map(
        (signal) => signal.watch().listen((_) async {
          await close();
          exit(0);
        }),
      )
      .toList();
  final timer = Timer.periodic(const Duration(seconds: 1), (_) async {
    if (busy || closing) return;
    if (idle.elapsed >= idleLimit) {
      await close();
      return;
    }
  });
  try {
    await for (final socket in server) {
      if (closing) {
        socket.destroy();
        continue;
      }
      busy = true;
      for (final key in compilers.keys.toList()) {
        if (compilers[key]!.idle.elapsed >= idleLimit) {
          await compilers.remove(key)!.close();
        }
      }
      idle.reset();
      String? key;
      final input = StreamIterator(
        socket
            .cast<List<int>>()
            .transform(utf8.decoder)
            .transform(const LineSplitter()),
      );
      try {
        final text = await input.moveNext().timeout(
          const Duration(seconds: 10),
        );
        if (!text) throw FormatException('empty resident request');
        final request = jsonDecode(input.current) as Map<String, dynamic>;
        final entry = request['entry'] as String;
        final output = request['output'] as String;
        final artifact = request['artifact'] as String;
        final identity = request['identity'] as String;
        final sources = request['sources'] as Map<String, dynamic>;
        if (!File(entry).isAbsolute ||
            !Directory(output).isAbsolute ||
            File(artifact).parent.path != '$output/.gart/kernel') {
          throw FormatException('invalid resident request paths');
        }
        key = entry;
        var compiler = compilers.remove(key);
        if (compiler != null &&
            (compiler.identity != identity ||
                compiler.exited ||
                request['reset'] == true)) {
          await compiler.close();
          compiler = null;
        }
        if (compiler == null) {
          if (compilers.length == 2) {
            final oldest = compilers.keys.first;
            await compilers.remove(oldest)!.close();
          }
          final scratch = directory.createTempSync('compiler-');
          final process = await Process.start('$sdk/bin/dartaotruntime', [
            '$sdk/bin/snapshots/frontend_server_aot.dart.snapshot',
            '--sdk-root',
            sdk,
            '--platform',
            File('$sdk/lib/_internal/vm_platform_strong.dill').uri.toString(),
            '--target=vm',
            if (sources.containsKey('$output/.dart_tool/native_assets.yaml'))
              '--native-assets=$output/.dart_tool/native_assets.yaml',
            '--incremental',
            '--enable-asserts',
            '--packages',
            '$output/.dart_tool/package_config.json',
            '--output-dill',
            '${scratch.path}/program.dill',
            '-Ddart.vm.product=false',
          ], workingDirectory: output);
          compiler = Compiler(process, scratch, identity, entry);
        }
        compilers[key] = compiler;
        final mode = compiler.requests == 0 ? 'cold' : 'incremental';
        final result = await compiler
            .compile(sources)
            .timeout(const Duration(seconds: 120));
        File(result).copySync(artifact);
        final response = {
          'mode': mode,
          'dependencies':
              compiler.dependencies.keys
                  .map((uri) => File.fromUri(Uri.parse(uri)).absolute.path)
                  .toSet()
                  .toList()
                ..sort(),
        };
        final rss = await memoryKiB(compiler.process.pid);
        if (rss == null ||
            rss > retainedRssKiB ||
            compiler.requests > editLimit) {
          compilers.remove(key);
          await compiler.close();
        }
        socket.writeln(jsonEncode(response));
      } catch (error) {
        if (key != null) await compilers.remove(key)?.close();
        socket.writeln(jsonEncode({'error': error.toString()}));
      } finally {
        try {
          await socket.flush();
          await socket.close();
        } on SocketException {
          socket.destroy();
        }
        await input.cancel();
        busy = false;
        idle.reset();
      }
    }
  } finally {
    timer.cancel();
    for (final subscription in signals) {
      await subscription.cancel();
    }
    await close();
  }
}
