import 'dart:async';
import 'dart:ffi';
import 'dart:io';
import 'dart:typed_data';

import 'file_handle.dart';
import 'native_errno.dart';
import 'slice.dart';
import 'string.dart';

final _libc = DynamicLibrary.process();
final _calloc = _libc
    .lookupFunction<
      Pointer<Void> Function(Size, Size),
      Pointer<Void> Function(int, int)
    >('calloc');
final _free = _libc
    .lookupFunction<Void Function(Pointer<Void>), void Function(Pointer<Void>)>(
      'free',
    );

abstract interface class ChildProcess {
  int get pid;
  Stream<List<int>> get stdout;
  Stream<List<int>> get stderr;
  Future<int> get exitCode;
  Future<int> write(Uint8List bytes);
  Future<void> closeInput();
  void closeOutput(bool diagnostic);
  // Native children use the Go signal number. Dart children use the signal object.
  bool kill(int number, ProcessSignal signal);
}

final class DartChild implements ChildProcess {
  final Process process;
  DartChild(this.process);

  @override
  int get pid => process.pid;
  @override
  Stream<List<int>> get stdout => process.stdout;
  @override
  Stream<List<int>> get stderr => process.stderr;
  @override
  Future<int> get exitCode => process.exitCode;
  @override
  Future<int> write(Uint8List bytes) async {
    process.stdin.add(bytes);
    await process.stdin.flush();
    return bytes.length;
  }

  @override
  Future<void> closeInput() => process.stdin.close();
  @override
  void closeOutput(bool diagnostic) {}
  @override
  bool kill(int number, ProcessSignal signal) => process.kill(signal);
}

final class _ProcessPipe {
  final NativePipeHandle handle;
  bool closed = false;
  _ProcessPipe(int descriptor) : handle = NativePipeHandle(descriptor);

  void close() {
    if (closed) return;
    closed = true;
    handle.closeSync();
  }

  Stream<List<int>> chunks() async* {
    final buffer = Uint8List(65536);
    try {
      while (!closed) {
        final count = await handle.read(buffer, 0, buffer.length);
        if (count == 0) return;
        yield Uint8List.sublistView(buffer, 0, count);
      }
    } on ClosedPipe {
      if (!closed) rethrow;
    } finally {
      close();
    }
  }
}

final class NativeChild implements ChildProcess {
  @override
  final int pid;
  final int _statusfd;
  final _ProcessPipe _input;
  final _ProcessPipe _output;
  final _ProcessPipe _diagnostic;
  Future<int>? _exit;
  bool _exited = false;

  NativeChild(NativeProcessResult result)
    : pid = result.pid,
      _statusfd = result.statusfd,
      _input = _ProcessPipe(result.input),
      _output = _ProcessPipe(result.output),
      _diagnostic = _ProcessPipe(result.diagnostic);

  static Future<NativeChild> start(
    Slice<GoString> arguments,
    GoString directory,
    Slice<GoString> environment,
  ) async {
    final allocations = <Pointer<NativeType>>[];
    Pointer<Uint8> string(GoString value) {
      final bytes = value.bytes;
      final pointer = _calloc(bytes.length + 1, 1).cast<Uint8>();
      if (pointer == nullptr) {
        throw ProcessException('', [], 'native process allocation', 12);
      }
      allocations.add(pointer);
      pointer.asTypedList(bytes.length).setAll(0, bytes);
      return pointer;
    }

    Pointer<Pointer<Uint8>> strings(Slice<GoString> values) {
      final pointer = _calloc(
        values.len + 1,
        sizeOf<Pointer<Uint8>>(),
      ).cast<Pointer<Uint8>>();
      if (pointer == nullptr) {
        throw ProcessException('', [], 'native process allocation', 12);
      }
      allocations.add(pointer);
      for (var index = 0; index < values.len; index++) {
        pointer[index] = string(values[index]);
      }
      return pointer;
    }

    try {
      final argv = strings(arguments);
      final result = nativeProcessStart(
        argv[0],
        argv,
        strings(environment),
        string(directory),
      );
      if (result.error != 0) {
        throw ProcessException(
          arguments[0].toDartString(),
          [],
          'native process launch',
          result.error,
        );
      }
      final child = NativeChild(result);
      final launch = _ProcessPipe(result.launch);
      var code = 0;
      try {
        final bytes = Uint8List(4);
        var count = 0;
        while (count < bytes.length) {
          final next = await launch.handle.read(bytes, count, bytes.length);
          if (next == 0) {
            if (count != 0) code = 5;
            break;
          }
          count += next;
        }
        if (count == bytes.length) {
          code = ByteData.sublistView(bytes).getInt32(0, Endian.host);
        }
      } finally {
        launch.close();
      }
      if (code != 0) {
        child._input.close();
        child._output.close();
        child._diagnostic.close();
        try {
          await child.exitCode;
        } on ProcessException catch (error) {
          // Another native reaper can consume a failed child first.
          if (error.errorCode != 10) rethrow;
        }
        throw ProcessException(
          arguments[0].toDartString(),
          [],
          'native child setup or exec',
          code,
        );
      }
      return child;
    } finally {
      for (final pointer in allocations) {
        _free(pointer.cast<Void>());
      }
    }
  }

  @override
  Stream<List<int>> get stdout => _output.chunks();
  @override
  Stream<List<int>> get stderr => _diagnostic.chunks();
  @override
  Future<int> write(Uint8List bytes) =>
      _input.handle.write(bytes, 0, bytes.length);
  @override
  Future<void> closeInput() async => _input.close();
  @override
  void closeOutput(bool diagnostic) =>
      (diagnostic ? _diagnostic : _output).close();

  @override
  Future<int> get exitCode => _exit ??= _wait();

  Future<int> _wait() async {
    try {
      while (true) {
        final result = nativeProcessWait(pid, _statusfd);
        if (result.error == 4) continue;
        if (result.error != 0) {
          throw ProcessException('$pid', [], 'native wait', result.error);
        }
        if (result.complete != 0) {
          _exited = true;
          final signal = result.value & 0x7f;
          return signal == 0 ? (result.value >> 8) & 0xff : -signal;
        }
        await Future<void>.delayed(const Duration(milliseconds: 1));
      }
    } finally {
      _exited = true;
      if (_statusfd >= 0) NativeFileHandle(_statusfd).closeSync();
    }
  }

  @override
  bool kill(int number, ProcessSignal signal) {
    if (_exited) return false;
    return nativeProcessKill(pid, _statusfd, number).error == 0;
  }
}
