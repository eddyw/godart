import 'dart:io';
import 'dart:typed_data';

import 'directory_descriptor.dart';
import 'file_handle.dart';
import 'file_stat.dart';
import 'removeall.dart';
import 'slice.dart';
import 'string.dart';
import 'syscall_directory.dart';

export 'directory_descriptor.dart';
export 'file_attributes.dart';
export 'file_handle.dart' show $nativePread, $nativePwrite;
export 'file_links.dart';
export 'file_readlink.dart';
export 'file_rename.dart';
export 'file_stat.dart';
export 'removeall.dart' show $syscallUnlink;
export 'syscall_directory.dart';
export 'working_directory.dart';

/// The name of the operating system, in the words of the Go constant GOOS.
GoString $goos() {
  if (Platform.isMacOS) return GoString.fromString('darwin');
  if (Platform.isLinux) return GoString.fromString('linux');
  if (Platform.isWindows) return GoString.fromString('windows');
  if (Platform.isAndroid) return GoString.fromString('android');
  if (Platform.isIOS) return GoString.fromString('ios');
  if (Platform.isFuchsia) return GoString.fromString('fuchsia');
  return GoString.fromString(Platform.operatingSystem);
}

/// The open files of the program, by handle.
///
/// A handle is an index into this list. A closed slot holds null, and a new
/// open call takes the first free slot.
final List<FileHandle?> _open = <FileHandle?>[];

/// The mode of an open call.
const int $openRead = 0;
const int $openWrite = 1;
const int $openAppend = 2;
const int $openReadWrite = 3;

/// The whence of a seek call.
const int $seekStart = 0;
const int $seekCurrent = 1;
const int $seekEnd = 2;

GoString _empty = GoString.empty;

/// _cause writes the message of a failure. Go reads such a text as the message
/// of an error value.
GoString _cause(Object e) {
  if (e is FileSystemException) {
    final String os = e.osError?.message ?? '';
    if (os.isNotEmpty) return GoString.fromString('${e.path}: $os');
    return GoString.fromString('${e.path}: ${e.message}');
  }
  return GoString.fromString(e.toString());
}

int _slot(FileHandle f) {
  for (int i = 0; i < _open.length; i++) {
    if (_open[i] == null) {
      _open[i] = f;
      return i;
    }
  }
  _open.add(f);
  return _open.length - 1;
}

int $fileFd(int handle, int std) {
  final ResourceHandle resource;
  switch (std) {
    case 1:
      resource = ResourceHandle.fromStdout(stdout);
    case 2:
      resource = ResourceHandle.fromStdout(stderr);
    case 3:
      resource = ResourceHandle.fromStdin(stdin);
    default:
      final file = _at(handle);
      if (file == null) return -1;
      return file.descriptor;
  }
  // SCM_RIGHTS data borrows the descriptor. Only the file owns its lifetime.
  final control = SocketControlMessage.fromHandles([resource]);
  return ByteData.sublistView(control.data).getInt32(0, Endian.host);
}

/// Opens a file and gives its handle. The second result holds the cause of a
/// failure, and it is empty after a success.
(int, GoString) $fileOpen(GoString path, int mode) {
  try {
    final RandomAccessFile f = File.fromRawPath(path.bytes)
        .openSync(mode: _mode(mode));
    // Dart holds no mode that reads and writes and keeps the position at the
    // start. FileMode.append keeps the content and stands at the end, thus the
    // open moves the position back.
    if (mode == $openReadWrite) f.setPositionSync(0);
    return (_slot(DartFileHandle(f)), _empty);
  } catch (e) {
    return (-1, _cause(e));
  }
}

FileMode _mode(int mode) {
  switch (mode) {
    case $openWrite:
      return FileMode.write;
    case $openAppend:
      return FileMode.writeOnlyAppend;
    case $openReadWrite:
      return FileMode.append;
  }
  return FileMode.read;
}

/// Reads bytes into the slice and gives the count. A count of 0 with no cause
/// means the end of the file.
(int, GoString, int) $fileRead(int handle, Slice<int> p) {
  final FileHandle? f = _at(handle);
  if (f == null) return (0, GoString.fromString('file already closed'), 0);
  if (p.len == 0) return (0, _empty, 0);
  try {
    final int n = f.readIntoSync(p.array, p.off, p.off + p.len);
    return (n, _empty, 0);
  } catch (e) {
    return (
      0,
      _cause(e),
      f is NativeFileHandle && e is FileSystemException
          ? e.osError?.errorCode ?? 0
          : 0,
    );
  }
}

/// Writes the bytes of the slice and gives the count.
(int, GoString, int) $fileWrite(int handle, Slice<int> p) {
  final FileHandle? f = _at(handle);
  if (f == null) return (0, GoString.fromString('file already closed'), 0);
  try {
    final count = f.writeFromSync(p.array, p.off, p.off + p.len);
    return (count, _empty, 0);
  } catch (e) {
    return (
      e is PartialFileWrite ? e.count : 0,
      _cause(e),
      f is NativeFileHandle && e is FileSystemException
          ? e.osError?.errorCode ?? 0
          : 0,
    );
  }
}

/// Moves the read position and gives the new offset from the start.
(int, GoString, int) $fileSeek(int handle, int offset, int whence) {
  final FileHandle? f = _at(handle);
  if (f == null) return (0, GoString.fromString('file already closed'), 0);
  if (f is NativeFileHandle) {
    final (position, code) = $syscallSeek(f.descriptor, offset, whence);
    return (code == 0 ? position : 0, _empty, code);
  }
  try {
    int target = offset;
    if (whence == $seekCurrent) target = f.positionSync() + offset;
    if (whence == $seekEnd) target = f.lengthSync() + offset;
    f.setPositionSync(target);
    return (target, _empty, 0);
  } catch (e) {
    return (0, _cause(e), 0);
  }
}

GoString $fileClose(int handle) {
  final FileHandle? f = _at(handle);
  if (f == null) return GoString.fromString('file already closed');
  _open[handle] = null;
  try {
    f.closeSync();
    return _empty;
  } catch (e) {
    return _cause(e);
  }
}

/// Cuts the file of the handle to the size, or it grows the file with zero
/// bytes. A file that no write mode opened gives a cause.
GoString $fileTruncate(int handle, int size) {
  final FileHandle? f = _at(handle);
  if (f == null) return GoString.fromString('file already closed');
  try {
    f.truncateSync(size);
    return _empty;
  } catch (e) {
    return _cause(e);
  }
}

GoString $fileSync(int handle) {
  final FileHandle? f = _at(handle);
  if (f == null) return GoString.fromString('file already closed');
  try {
    f.flushSync();
    return _empty;
  } catch (e) {
    return _cause(e);
  }
}

FileHandle? _at(int handle) {
  if (handle < 0 || handle >= _open.length) return null;
  return _open[handle];
}

/// Reads a whole file.
(Slice<int>, GoString) $readFile(GoString path) {
  try {
    return (Slice.of(File.fromRawPath(path.bytes).readAsBytesSync()), _empty);
  } catch (e) {
    return (Slice<int>.nil(), _cause(e));
  }
}

/// Reads the metadata of a path. The 3 values are the size in bytes, the
/// directory flag, and the time of the last change in nanoseconds.
(int, bool, int, GoString) $stat(GoString path) {
  final String name = path.toDartString();
  try {
    final FileStat s = FileStat.statSync(name);
    if (s.type == FileSystemEntityType.notFound) {
      return (
        0,
        false,
        0,
        GoString.fromString('$name: no such file or directory'),
      );
    }
    final bool dir = s.type == FileSystemEntityType.directory;
    return (s.size, dir, s.modified.microsecondsSinceEpoch * 1000, _empty);
  } catch (e) {
    return (0, false, 0, _cause(e));
  }
}

/// Reads the metadata of a path, and it follows no symbolic link. The 4 values
/// are the size in bytes, the directory flag, the link flag, and the time of
/// the last change in nanoseconds.
///
/// The metadata of a link itself is out of reach in dart:io, thus the size of a
/// link is the length of its target, and the time of a link is 0.
(int, bool, bool, int, GoString) $lstat(GoString path) {
  final String name = path.toDartString();
  try {
    final FileSystemEntityType t = FileSystemEntity.typeSync(
      name,
      followLinks: false,
    );
    if (t == FileSystemEntityType.notFound) {
      return (
        0,
        false,
        false,
        0,
        GoString.fromString('$name: no such file or directory'),
      );
    }
    if (t == FileSystemEntityType.link) {
      final String target = Link(name).targetSync();
      return (target.length, false, true, 0, _empty);
    }
    final FileStat s = FileStat.statSync(name);
    final bool dir = s.type == FileSystemEntityType.directory;
    return (
      s.size,
      dir,
      false,
      s.modified.microsecondsSinceEpoch * 1000,
      _empty,
    );
  } catch (e) {
    return (0, false, false, 0, _cause(e));
  }
}

/// Makes a symbolic link at the path, and the link points at the target.
GoString $symlink(GoString target, GoString link) {
  try {
    Link(link.toDartString()).createSync(target.toDartString());
    return _empty;
  } catch (e) {
    return _cause(e);
  }
}

/// Runs 1 command of the platform and gives the cause of a failure.
///
/// dart:io holds no call for a hard link, for a permission bit, and for an
/// owner. The command of the platform is the only road to the 3, and it writes
/// the message of the system on standard error.
GoString _run(String program, List<String> args, String path) {
  try {
    final ProcessResult r = Process.runSync(program, args);
    if (r.exitCode == 0) return _empty;
    final String text = '${r.stderr}'.trim();
    if (text.isEmpty) {
      return GoString.fromString(
        '$path: $program gave the status ${r.exitCode}',
      );
    }
    return GoString.fromString(text);
  } catch (e) {
    return _cause(e);
  }
}

/// Makes a hard link at the path link, and it names the content of target.
GoString $link(GoString target, GoString link) {
  final String to = link.toDartString();
  return _run('ln', <String>['--', target.toDartString(), to], to);
}

/// _umask holds the file mode creation mask of the process, or -1 before the
/// first read.
int _umaskValue = -1;

/// The file mode creation mask of the process.
///
/// A create of Go gives the file the permission bits of the call without the
/// bits of this mask. dart:io writes no mode at a create, thus the caller
/// applies the mask itself and then runs chmod. The shell holds the only read
/// of the mask that a process outside libc can take, thus the value comes from
/// 1 run of it and it stays.
int $umask() {
  if (_umaskValue >= 0) return _umaskValue;
  _umaskValue = 0;
  try {
    final ProcessResult r = Process.runSync('sh', <String>['-c', 'umask']);
    if (r.exitCode == 0) {
      final int? v = int.tryParse('${r.stdout}'.trim(), radix: 8);
      if (v != null) _umaskValue = v & 0xFFF;
    }
  } catch (_) {
    // A host with no shell keeps the mask 0, thus a create writes the bits of
    // the call itself.
  }
  return _umaskValue;
}

/// Writes the permission bits of a path. The mode holds the 12 bits of a POSIX
/// mode, thus the text of the command carries 4 octal digits.
GoString $chmod(GoString path, int mode) {
  final String name = path.toDartString();
  final String bits = (mode & 0xFFF).toRadixString(8).padLeft(4, '0');
  return _run('chmod', <String>['--', bits, name], name);
}

/// Writes the owner and the group of a path. A number below 0 leaves that 1 of
/// the 2 alone, and follow false takes a symbolic link itself.
GoString $chown(GoString path, int uid, int gid, bool follow) {
  final String name = path.toDartString();
  if (uid < 0 && gid < 0) {
    // The call changes nothing, and the caller still needs the report of a path
    // that does not exist.
    final FileSystemEntityType t = FileSystemEntity.typeSync(
      name,
      followLinks: follow,
    );
    if (t == FileSystemEntityType.notFound) {
      return GoString.fromString('$name: no such file or directory');
    }
    return _empty;
  }
  final String owner = uid < 0 ? ':$gid' : (gid < 0 ? '$uid' : '$uid:$gid');
  final List<String> args = follow
      ? <String>['--', owner, name]
      : <String>['-h', '--', owner, name];
  return _run('chown', args, name);
}

/// Writes the time of the last read and the time of the last change, in
/// nanoseconds since the start of the Unix epoch.
///
/// dart:io holds the millisecond, thus the call drops the finer part.
GoString $chtimes(
  GoString path,
  int atime,
  bool setAtime,
  int mtime,
  bool setMtime,
) {
  final String name = path.toDartString();
  if (!setAtime && !setMtime) {
    // The call writes no time, and the caller still needs the report of a path
    // that does not exist.
    if (FileSystemEntity.typeSync(name) == FileSystemEntityType.notFound) {
      return GoString.fromString('$name: no such file or directory');
    }
    return _empty;
  }
  try {
    final File f = File(name);
    if (setAtime) {
      f.setLastAccessedSync(DateTime.fromMicrosecondsSinceEpoch(atime ~/ 1000));
    }
    if (setMtime) {
      f.setLastModifiedSync(DateTime.fromMicrosecondsSinceEpoch(mtime ~/ 1000));
    }
    return _empty;
  } catch (e) {
    return _cause(e);
  }
}

/// Gives the target of 1 symbolic link. A path that holds no link gives a
/// cause.
(GoString, GoString) $readLink(GoString path) {
  final String name = path.toDartString();
  try {
    if (FileSystemEntity.typeSync(name, followLinks: false) !=
        FileSystemEntityType.link) {
      return (_empty, GoString.fromString('invalid argument'));
    }
    return (GoString.fromString(Link(name).targetSync()), _empty);
  } catch (e) {
    return (_empty, _cause(e));
  }
}

/// Gives the path after each symbolic link, from the root. The path must exist.
(GoString, GoString) $resolveSymlinks(GoString path) {
  try {
    final String out = File(path.toDartString()).resolveSymbolicLinksSync();
    return (GoString.fromString(out), _empty);
  } catch (e) {
    return (_empty, _cause(e));
  }
}

(int, GoString) $remove(GoString path) {
  if (hasDescriptorRemoval) return (removeDescriptors(path), _empty);
  final String name = path.toDartString();
  try {
    final FileSystemEntityType t = FileSystemEntity.typeSync(
      name,
      followLinks: false,
    );
    if (t == FileSystemEntityType.notFound) {
      return (0, GoString.fromString('$name: no such file or directory'));
    }
    if (t == FileSystemEntityType.link) {
      Link(name).deleteSync();
      return (0, _empty);
    }
    if (t == FileSystemEntityType.directory) {
      Directory(name).deleteSync();
      return (0, _empty);
    }
    File(name).deleteSync();
    return (0, _empty);
  } catch (e) {
    return (0, _cause(e));
  }
}

(GoString, GoString, int, GoString) $removeAll(GoString path) {
  if (hasDescriptorRemoval) {
    final (op, name, errno) = removeAllDescriptors(path);
    return (op, name, errno, _empty);
  }
  if (path.bytes.contains(0)) {
    return (GoString.fromString('RemoveAll'), path, 22, _empty);
  }
  final name = path.toDartString();
  (GoString, GoString, int, GoString)? first;
  try {
    final type = FileSystemEntity.typeSync(name, followLinks: false);
    if (type == FileSystemEntityType.notFound) {
      return (_empty, _empty, 0, _empty);
    }
    if (type == FileSystemEntityType.directory) {
      try {
        for (final child in Directory(name).listSync(followLinks: false)) {
          final result = $removeAll(GoString.fromString(child.path));
          if (result.$4.length != 0) first ??= result;
        }
      } catch (e) {
        first = (GoString.fromString('readdirnames'), path, 0, _cause(e));
      }
      Directory(name).deleteSync();
    } else if (type == FileSystemEntityType.link) {
      Link(name).deleteSync();
    } else {
      File(name).deleteSync();
    }
    return (_empty, _empty, 0, _empty);
  } catch (e) {
    return first ?? (GoString.fromString('removeall'), path, 0, _cause(e));
  }
}

GoString $mkdir(GoString path) {
  try {
    Directory(path.toDartString()).createSync();
    return _empty;
  } catch (e) {
    return _cause(e);
  }
}

GoString $rename(GoString from, GoString to) {
  try {
    final source = from.toDartString();
    final target = to.toDartString();
    switch (FileSystemEntity.typeSync(source, followLinks: false)) {
      case FileSystemEntityType.directory:
        Directory(source).renameSync(target);
      case FileSystemEntityType.link:
        Link(source).renameSync(target);
      default:
        File(source).renameSync(target);
    }
    return _empty;
  } catch (e) {
    return _cause(e);
  }
}

(bool, GoString) $fileIdentical(GoString first, GoString second) {
  try {
    return (
      FileSystemEntity.identicalSync(
        first.toDartString(),
        second.toDartString(),
      ),
      _empty,
    );
  } catch (e) {
    return (false, _cause(e));
  }
}

/// Reads the names inside a directory. Each name holds no directory part.
(Slice<GoString>, GoString) $readDir(GoString path) {
  try {
    if ($hasNativeStat) {
      final names = readDirectoryNames(path);
      names.sort((GoString a, GoString b) => $compare(a, b));
      return (Slice.of(names), _empty);
    }
    final List<FileSystemEntity> list = Directory(path.toDartString())
        .listSync(followLinks: false);
    final List<GoString> names = <GoString>[];
    for (final FileSystemEntity e in list) {
      names.add(
        GoString.fromString(
          e.uri.pathSegments.where((String s) => s.isNotEmpty).last,
        ),
      );
    }
    names.sort((GoString a, GoString b) => $compare(a, b));
    return (Slice.of(names), _empty);
  } catch (e) {
    return (Slice<GoString>.nil(), _cause(e));
  }
}

/// Reads 1 variable of the environment. The flag is false when the environment
/// holds no such name.
(GoString, bool) $getenv(GoString name) {
  final String? v = Platform.environment[name.toDartString()];
  if (v == null) return (_empty, false);
  return (GoString.fromString(v), true);
}

/// Reads each variable of the environment, in the form name=value.
Slice<GoString> $environ() {
  final List<GoString> out = <GoString>[];
  Platform.environment.forEach((String k, String v) {
    out.add(GoString.fromString('$k=$v'));
  });
  out.sort((GoString a, GoString b) => $compare(a, b));
  return Slice.of(out);
}

(GoString, GoString) $getwd() {
  try {
    return (GoString.fromString(Directory.current.path), _empty);
  } catch (e) {
    return (_empty, _cause(e));
  }
}

/// Gives the process id of the program.
int $getpid() => pid;

/// Gives the name of the host.
(GoString, GoString) $hostname() {
  try {
    return (GoString.fromString(Platform.localHostname), _empty);
  } catch (e) {
    return (_empty, _cause(e));
  }
}

/// Gives the path of the program that runs. The Dart VM runs the emitted Dart,
/// thus the path names the VM.
(GoString, GoString) $executable() {
  try {
    return (GoString.fromString(Platform.resolvedExecutable), _empty);
  } catch (e) {
    return (_empty, _cause(e));
  }
}

GoString $chdir(GoString path) {
  try {
    Directory.current = Directory(path.toDartString());
    return _empty;
  } catch (e) {
    return _cause(e);
  }
}

GoString $tempDir() => GoString.fromString(Directory.systemTemp.path);

GoString $pathSeparator() => GoString.fromString(Platform.pathSeparator);

/// Writes text to standard error.
void $stderr(GoString text) => stderr.add(text.bytes);

/// Reads bytes from standard input. A count of 0 means the end of the input.
///
/// The read stops at a newline. A terminal gives 1 line for each read, thus the
/// rule matches Go there. A pipe gives Go the whole buffer, and it gives this
/// function 1 line.
(int, GoString) $stdinRead(Slice<int> p) {
  if (p.len == 0) return (0, _empty);
  try {
    int n = 0;
    while (n < p.len) {
      final int b = stdin.readByteSync();
      if (b < 0) break;
      p.array[p.off + n] = b;
      n++;
      if (b == 10) break;
    }
    return (n, _empty);
  } catch (e) {
    return (0, _cause(e));
  }
}

(int, int, int) $fileOpenFlags(GoString path, int flags, int mode) {
  final (file, code) = NativeFileHandle.open(path, flags, mode);
  if (code != 0) return (-1, -1, code);
  final opened = file!;
  final (metadata, statCode) = $nativeFstat(opened.descriptor);
  if (statCode != 0) {
    $syscallClose(opened.descriptor);
    return (-1, -1, statCode);
  }
  if (metadata.array[metadata.off + 1] & 0xf000 == 0x4000) {
    return (-1, opened.descriptor, 0);
  }
  return (_slot(opened), -1, 0);
}
