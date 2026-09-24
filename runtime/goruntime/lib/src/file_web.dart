/// The web form of file.dart.
///
/// A web target holds no file access. This file still declares each name of
/// file.dart, because the Go package os names each 1. A program that only
/// reads an environment value would otherwise fail in the Dart compile, and
/// the whole web target would stay out of reach.
///
/// Each operation that the web cannot do gives the cause "function not
/// implemented". The package os wraps that text in a PathError, thus the
/// program reads the same error that Go gives under GOOS=js.
library;

import 'goint.dart';
import 'platform_web.dart';
import 'slice.dart';
import 'string.dart';

/// The cause of each operation that the web target does not hold. Go gives
/// the same text for the errno ENOSYS.
final GoString _noSys = GoString.fromString('function not implemented');

/// The name of the operating system. The web target runs under a JS host,
/// and Go names that system js.
GoString $goos() => GoString.fromString('js');

(int, GoString) $fileOpen(GoString path, int mode) => (0, _noSys);

int $fileFd(int handle, int std) =>
    throw UnsupportedError('native file descriptors');

(int, int) $directoryOpen(GoString path) => (-1, 38);

int $directoryFd(int fd) =>
    throw UnsupportedError('native directory descriptors');

int $directoryClose(int fd) => 38;

int $directoryChdir(int fd) => 38;

int $directoryCreate(GoString path, int mode) => 38;

(int, GoString) $fileRead(int handle, Slice<GoInt> p) => (0, _noSys);

(int, GoString) $fileWrite(int handle, Slice<GoInt> p) => (0, _noSys);

(int, GoString) $fileSeek(int handle, int offset, int whence) => (0, _noSys);

GoString $fileClose(int handle) => _noSys;

GoString $fileTruncate(int handle, int size) => _noSys;

GoString $fileSync(int handle) => _noSys;

(Slice<int>, GoString) $readFile(GoString path) => (Slice<int>.nil(), _noSys);

GoString $writeFile(GoString path, Slice<GoInt> data) => _noSys;

const bool $hasNativeStat = false;

int $nativeRename(GoString from, GoString to) => 38;

(Slice<int>, int) $nativeStat(GoString path, bool follow) =>
    (Slice<int>.nil(), 38);

(Slice<int>, int) $nativeFstat(int fd) => (Slice<int>.nil(), 38);

(int, bool, int, GoString) $stat(GoString path) => (0, false, 0, _noSys);

(int, bool, bool, int, GoString) $lstat(GoString path) =>
    (0, false, false, 0, _noSys);

GoString $symlink(GoString target, GoString link) => _noSys;

GoString $link(GoString target, GoString link) => _noSys;

int $fileChmod(int fd, int mode) => 38;

int $fileChown(int fd, int uid, int gid) => 38;

int $nativeChmod(GoString path, int mode) => 38;

int $nativeChown(GoString path, int uid, int gid, bool follow) => 38;

GoString $chmod(GoString path, int mode) => _noSys;

/// The file mode creation mask of the process. A browser holds no such mask,
/// and it writes no file, thus the value 0 leaves each mode of a call whole.
int $umask() => 0;

GoString $chown(GoString path, int uid, int gid, bool follow) => _noSys;

GoString $chtimes(
  GoString path,
  int atime,
  bool setAtime,
  int mtime,
  bool setMtime,
) => _noSys;

(GoString, GoString) $readLink(GoString path) => (GoString.empty, _noSys);

(GoString, GoString) $resolveSymlinks(GoString path) =>
    (GoString.empty, _noSys);

GoString $remove(GoString path) => _noSys;

(GoString, GoString, GoString) $removeAll(GoString path) =>
    (GoString.fromString('removeall'), path, _noSys);

GoString $mkdir(GoString path) => _noSys;

GoString $rename(GoString from, GoString to) => _noSys;

(bool, GoString) $fileIdentical(GoString first, GoString second) =>
    (false, _noSys);

(Slice<GoString>, GoString) $readDir(GoString path) =>
    (Slice<GoString>.nil(), _noSys);

/// The environment of the web target is empty. A browser holds no such table,
/// and Go gives the same answer under GOOS=js.
(GoString, bool) $getenv(GoString name) => (GoString.empty, false);

Slice<GoString> $environ() => Slice<GoString>.nil();

/// The working directory of the web target. Go gives the root under GOOS=js,
/// because the file system of that target starts there.
(GoString, GoString) $getwd() => (GoString.fromString('/'), GoString.empty);

/// The process identifier. A web host holds no process, and Go gives 0.
int $getpid() => 0;

(GoString, GoString) $hostname() => (GoString.empty, _noSys);

(GoString, GoString) $executable() => (GoString.empty, _noSys);

GoString $chdir(GoString path) => _noSys;

GoString $tempDir() => GoString.fromString('/tmp');

GoString $pathSeparator() => GoString.fromString('/');

/// Writes text to standard error. The web target holds no such stream, thus
/// the text reaches the console of the host.
void $stderr(GoString text) => $platformStderrText(text.toDartString());

/// Reads bytes of standard input. A web target holds no standard input, thus
/// the read reports the end of the input at once.
///
/// Each program that reads the package fmt names this function, because the
/// scan family of that package reads standard input.
(int, GoString) $stdinRead(Slice<GoInt> p) => (0, GoString.empty);

int $nativeChtimes(
  GoString path,
  int atime,
  bool setAtime,
  int mtime,
  bool setMtime,
) => 38;

(GoString, int) $nativeReadlink(GoString path) => (GoString.empty, 38);

int $nativeLink(GoString from, GoString to) => 38;
int $nativeSymlink(GoString from, GoString to) => 38;
