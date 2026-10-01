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

(int, int, int) $fileFromFD(int fd) =>
    throw UnsupportedError('native file descriptors');

(int, int) $directoryOpen(GoString path) => (-1, 38);

int $directoryFd(int fd) =>
    throw UnsupportedError('native directory descriptors');

int $directoryClose(int fd) => 38;

int $directoryChdir(int fd) => 38;

int $nativeChdir(GoString path) => 38;

(GoString, int) $nativeGetwd() => (GoString.empty, 38);

int $directoryCreate(GoString path, int mode) => 38;

(int, GoString, int) $fileRead(int handle, Slice<GoInt> p) => (0, _noSys, 0);

(int, GoString, int) $fileWrite(int handle, Slice<GoInt> p) => (0, _noSys, 0);

(int, GoString, int) $fileSeek(int handle, int offset, int whence) =>
    (0, _noSys, 0);

(int, GoString) $fileClose(int handle) => (38, _noSys);

GoString $fileTruncate(int handle, int size) => _noSys;

GoString $fileSync(int handle) => _noSys;

(Slice<int>, GoString) $readFile(GoString path) => (Slice<int>.nil(), _noSys);

const bool $hasNativeStat = false;

int $nativeExecutableAccess(GoString path) =>
    throw UnsupportedError('native executable access');

int $nativeRename(GoString from, GoString to) => 38;

(Slice<int>, int) $nativeStat(GoString path, bool follow) =>
    (Slice<int>.nil(), 38);

(Slice<int>, int) $nativeFstat(int fd) => (Slice<int>.nil(), 38);

(int, bool, int, GoString) $stat(GoString path) => (0, false, 0, _noSys);

(int, bool, bool, int, GoString) $lstat(GoString path) =>
    (0, false, false, 0, _noSys);

GoString $symlink(GoString target, GoString link) => _noSys;

GoString $link(GoString target, GoString link) => _noSys;

int $nativeTruncate(GoString path, int size) => 38;

(int, int) $nativePread(int fd, Slice<GoInt> bytes, int offset) => (0, 38);

(int, int) $nativePwrite(int fd, Slice<GoInt> bytes, int offset) => (0, 38);

int $nativeFtruncate(int fd, int size) => 38;

int $nativeFsync(int fd) => 38;

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

(int, GoString) $remove(GoString path) => (0, _noSys);

(GoString, GoString, int, GoString) $removeAll(GoString path) =>
    path.bytes.contains(0)
    ? (GoString.fromString('RemoveAll'), path, 22, GoString.empty)
    : (GoString.fromString('removeall'), path, 0, _noSys);

GoString $mkdir(GoString path) => _noSys;

GoString $rename(GoString from, GoString to) => _noSys;

(bool, GoString) $fileIdentical(GoString first, GoString second) =>
    (false, _noSys);

(Slice<GoString>, GoString) $readDir(GoString path) =>
    (Slice<GoString>.nil(), _noSys);

/// Node supplies the initial environment. A browser has no environment table.
(GoString, bool) $getenv(GoString name) {
  final value = $platformEnvironment[name.toDartString()];
  return value == null
      ? (GoString.empty, false)
      : (GoString.fromString(value), true);
}

Slice<GoString> $environ() => Slice<GoString>.of([
  for (final entry in $platformEnvironment.entries)
    GoString.fromString('${entry.key}=${entry.value}'),
]);

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

void $stderr(GoString text) => $platformStderr(text.bytes);

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

(int, int) $syscallOpen(GoString path, int flags, int mode) => (-1, 38);
(int, int) $syscallSeek(int fd, int offset, int whence) => (-1, 38);

int $syscallClose(int fd) => 38;

(int, int) $syscallDup(int fd) => (-1, 38);

int $syscallDup2(int oldfd, int newfd) => 38;

int $syscallDup3(int oldfd, int newfd, int flags) => 38;

int $syscallFsync(int fd) => 38;

int $syscallAccess(GoString path, int mode) => 38;

int $syscallFchmodat(int fd, GoString path, int mode, int flags) => 38;

int $syscallFdatasync(int fd) => 38;

(int, int, int) $syscallGetrlimit(int resource, bool output) => (0, 0, 38);

int $syscallSetrlimit(int resource, bool input, int cur, int max) => 38;

(int, int) $syscallFcntl(int fd, int cmd, int arg) => (-1, 38);
(int, int) $syscallReadDirent(int fd, Slice<GoInt> data) => (0, 38);

(int, int, int) $fileOpenFlags(GoString path, int flags, int mode) =>
    (-1, -1, 38);

(Slice<int>, int) $nativeLstatat(int fd, GoString path) =>
    (Slice<int>.nil(), 38);

(GoString, int) $syscallGetwd() => (GoString.empty, 38);

int $syscallUnlink(GoString path, bool directory) => 38;

(int, int) $syscallReadlink(GoString path, Slice<GoInt> bytes) => (-1, 38);

(int, int) $rootOpen(GoString path) => (-1, 38);
(int, int) $rootOpenAt(int parent, GoString path) => (-1, 38);
(int, int) $rootOpenDir(int parent, GoString path) => (-1, 38);
(int, int, int) $rootOpenFile(int parent, GoString path, int flags, int mode) =>
    (-1, -1, 38);
int $rootMkdir(int parent, GoString path, int mode) => 38;
(GoString, int) $rootReadlink(int parent, GoString path) =>
    (GoString.empty, 38);
int $rootChmod(int parent, GoString path, int mode) => 38;
int $rootChown(int parent, GoString path, int uid, int gid) => 38;
int $rootChtimes(
  int parent,
  GoString path,
  int atime,
  bool setAtime,
  int mtime,
  bool setMtime,
) => 38;
int $rootLink(int oldParent, GoString from, int newParent, GoString to) => 38;
int $rootRename(int oldParent, GoString from, int newParent, GoString to) => 38;
int $rootSymlink(GoString target, int parent, GoString path) => 38;
int $rootRemove(int parent, GoString path) => 38;
int $rootRemoveAll(int parent, GoString path) => 38;

(Slice<int>, int) $syscallGetrusage(int who, bool output) =>
    (Slice<int>.nil(), 38);

int $syscallUmask(int mask) =>
    throw UnsupportedError('native process file creation mask');

int $syscallFaccessat(int fd, GoString path, int mode, int flags) => 38;

(int, int, int, int) $syscallFaccessatStat(int fd, GoString path, int flags) =>
    (0, 0, 0, 38);

bool $syscallCapDacOverride() => false;

int $syscallMkdirat(int fd, GoString path, int mode) => 38;

int $syscallFchownat(int fd, GoString path, int uid, int gid, int flags) => 38;

int $syscallUnlinkat(int fd, GoString path) => 38;

int $syscallRenameat(
  int oldFD,
  GoString oldPath,
  int newFD,
  GoString newPath,
) => 38;
