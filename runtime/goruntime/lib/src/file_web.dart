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

(int, int, int) $filePipe() => (-1, -1, -1);

(int, int, int) $fileFromFD(int fd) =>
    throw UnsupportedError('native file descriptors');

(int, int) $directoryOpen(GoString path) => (-1, 38);

int $directoryFd(int fd) =>
    throw UnsupportedError('native directory descriptors');

int $directoryClose(int fd) => 38;

int $directoryChdir(int fd) => -1;

int $nativeChdir(GoString path) => -1;

(GoString, int) $nativeGetwd() => (GoString.empty, 38);

int $directoryCreate(GoString path, int mode) => -1;

(int, GoString, int, bool) $fileRead(int handle, Slice<GoInt> p) =>
    (0, _noSys, 0, false);

(int, GoString, int, bool) $fileWrite(int handle, Slice<GoInt> p) =>
    (0, _noSys, 0, false);

bool $fileDeadline(int handle, int timeout, int mode, bool clear) => false;

(int, GoString, int) $fileSeek(int handle, int offset, int whence) =>
    (0, _noSys, 0);

(int, GoString) $fileClose(int handle) => (38, _noSys);

GoString $fileTruncate(int handle, int size) => _noSys;

GoString $fileSync(int handle) => _noSys;

(Slice<int>, GoString) $readFile(GoString path) => (Slice<int>.nil(), _noSys);

const bool $hasNativeStat = false;

int $nativeExecutableAccess(GoString path) =>
    throw UnsupportedError('native executable access');

int $nativeRename(GoString from, GoString to) => -1;

(Slice<int>, int) $nativeStat(GoString path, bool follow) =>
    (Slice<int>.nil(), -1);

(Slice<int>, int) $nativeFstat(int fd) => (Slice<int>.nil(), -1);

(int, bool, int, GoString) $stat(GoString path) => (0, false, 0, _noSys);

(int, bool, bool, int, GoString) $lstat(GoString path) =>
    (0, false, false, 0, _noSys);

GoString $symlink(GoString target, GoString link) => _noSys;

GoString $link(GoString target, GoString link) => _noSys;

int $nativeTruncate(GoString path, int size) => -1;

(int, int) $nativePread(int fd, Slice<GoInt> bytes, int offset) => (0, -1);

(int, int) $nativePwrite(int fd, Slice<GoInt> bytes, int offset) => (0, -1);

int $nativeFtruncate(int fd, int size) => -1;

int $nativeFsync(int fd) => 38;

int $fileChmod(int fd, int mode) => -1;

int $fileChown(int fd, int uid, int gid) => -1;

int $nativeChmod(GoString path, int mode) => -1;

int $nativeChown(GoString path, int uid, int gid, bool follow) => -1;

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

int $nativeLink(GoString from, GoString to) => -1;
int $nativeSymlink(GoString from, GoString to) => -1;

(int, int) $syscallSocket(int domain, int type, int protocol) => (-1, -1);

(int, int) $syscallOpen(GoString path, int flags, int mode) => (-1, -1);

(int, int) $syscallOpenat(int parent, GoString path, int flags, int mode) =>
    (-1, 38);
(int, int) $syscallSeek(int fd, int offset, int whence) => (-1, -1);

int $syscallClose(int fd) => -1;

(int, int) $syscallDup(int fd) => (-1, -1);

int $syscallDup2(int oldfd, int newfd) => -1;

int $syscallDup3(int oldfd, int newfd, int flags) => 38;

int $syscallFsync(int fd) => -1;

int $syscallAccess(GoString path, int mode) => -1;

int $syscallFchmodat(int fd, GoString path, int mode, int flags) => 38;

int $syscallFdatasync(int fd) => 38;

(Slice<int>, int) $syscallUname(bool output) => (Slice<int>.nil(), -1);

int $syscallMkfifo(GoString path, int mode) => -1;

int $syscallChflags(GoString path, int flags) => -1;

int $syscallFchflags(int fd, int flags) => -1;

(int, int) $syscallPathconf(GoString path, int name) => (0, -1);

(int, int) $syscallFpathconf(int fd, int name) => (0, -1);

(Slice<int>, int) $syscallRouteRIB(int facility, int param) =>
    (Slice<int>.nil(), -1);

(GoString, int) $syscallSysctl(GoString name) => (GoString.empty, -1);

(int, int) $syscallSysctlUint32(GoString name) => (0, -1);

(Slice<int>, int) $syscallStatfs(GoString path, bool output) =>
    (Slice<int>.nil(), -1);

(Slice<int>, int) $syscallFstatfs(int fd, bool output) =>
    (Slice<int>.nil(), -1);

(int, Slice<int>, int) $syscallGetfsstat(int count, int flags) =>
    (-1, Slice<int>.nil(), -1);

(int, int, int) $syscallGettimeofday(bool output) => (0, 0, -1);

(int, int, int) $syscallGetrlimit(int resource, bool output) => (0, 0, -1);

int $syscallSetrlimit(int resource, bool input, int cur, int max) => -1;

(Slice<int>, int) $syscallFcntlFlock(
  int fd,
  int cmd,
  bool input,
  int start,
  int length,
  int pid,
  int kind,
  int whence,
  int pad0,
  int pad1,
) => (Slice<int>.nil(), -1);

(int, int) $syscallFcntl(int fd, int cmd, int arg) => (-1, -1);
(int, int) $syscallReadDirent(int fd, Slice<GoInt> data) => (0, -1);

(int, int, int) $fileOpenFlags(GoString path, int flags, int mode) =>
    (-1, -1, 38);

(Slice<int>, int) $nativeFstatat(
  int fd,
  GoString path,
  bool output,
  int flags,
) => (Slice<int>.nil(), path.bytes.contains(0) ? 22 : -1);

(Slice<int>, int) $nativeLstatat(int fd, GoString path) =>
    (Slice<int>.nil(), 38);

(GoString, int) $syscallGetwd() => (GoString.empty, -1);

(int, int) $syscallGetcwd(Slice<GoInt> data) => (-1, 38);

int $syscallUtimensat(
  int fd,
  GoString path,
  int atimeSec,
  int atimeNsec,
  int mtimeSec,
  int mtimeNsec,
) => -1;

int $syscallUtimes(
  GoString path,
  int atimeSec,
  int atimeUsec,
  int mtimeSec,
  int mtimeUsec,
) => -1;

int $syscallFutimesat(
  int fd,
  GoString path,
  int atimeSec,
  int atimeUsec,
  int mtimeSec,
  int mtimeUsec,
) => 38;

int $syscallFutimes(
  int fd,
  int atimeSec,
  int atimeUsec,
  int mtimeSec,
  int mtimeUsec,
) => -1;

int $syscallUtime(GoString path, bool hasTimes, int atime, int mtime) => 38;

int $syscallUnlink(GoString path, bool directory) => -1;

(int, int) $syscallReadlink(GoString path, Slice<GoInt> bytes) => (-1, -1);

(int, int) $rootOpen(GoString path) => (-1, -1);
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
    (Slice<int>.nil(), -1);

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

(int, int) $syscallGetxattr(GoString path, GoString attr, Slice<GoInt> dest) =>
    (0, -1);
(int, int) $syscallListxattr(GoString path, Slice<GoInt> dest) => (0, -1);
int $syscallSetxattr(
  GoString path,
  GoString attr,
  Slice<GoInt> data,
  int flags,
) => -1;
int $syscallRemovexattr(GoString path, GoString attr) => -1;

(int, int) $nativeRead(int fd, Slice<GoInt> bytes) => (-1, -1);

(int, int) $nativeWrite(int fd, Slice<GoInt> bytes) => (-1, -1);

int $syscallAttachLsf(int fd, Slice<GoInt> data) => 38;

int $syscallSetsockoptString(int fd, int level, int option, GoString value) =>
    -1;

(int, int) $syscallGetsockoptInt(int fd, int level, int option) => (0, -1);

int $syscallSetsockoptInt(int fd, int level, int option, int value) => -1;

(int, int, int) $syscallSocketpair(int domain, int type, int protocol) =>
    (0, 0, -1);

(int, int) $syscallGetsockoptByte(int fd, int level, int option) => (0, -1);

(int, int, int, int) $syscallGetsockoptUcred(int fd, int level, int option) =>
    (0, 0, 0, 38);

int $syscallGetsockoptIPMreq(
  int fd,
  int level,
  int option,
  Slice<GoInt> data,
) => -1;

int $syscallSetsockoptIPMreq(
  int fd,
  int level,
  int option,
  Slice<GoInt> data,
) => -1;

int $syscallGetsockoptIPMreqn(
  int fd,
  int level,
  int option,
  Slice<GoInt> data,
) => 38;

int $syscallSetsockoptIPMreqn(
  int fd,
  int level,
  int option,
  Slice<GoInt> data,
) => 38;

int $syscallGetsockoptIPv6Mreq(
  int fd,
  int level,
  int option,
  Slice<GoInt> data,
) => -1;

int $syscallSetsockoptIPv6Mreq(
  int fd,
  int level,
  int option,
  Slice<GoInt> data,
) => -1;

int $syscallSetsockoptTimeval(
  int fd,
  int level,
  int option,
  Slice<GoInt> data,
) => -1;

int $syscallGetsockoptICMPv6Filter(
  int fd,
  int level,
  int option,
  Slice<GoInt> data,
) => -1;

int $syscallSetsockoptICMPv6Filter(
  int fd,
  int level,
  int option,
  Slice<GoInt> data,
) => -1;

(Slice<int>, int) $syscallMmap(
  int fd,
  int offset,
  int length,
  int prot,
  int flags,
) => (Slice<int>.nil(), -1);

int $syscallMadvise(Slice<GoInt> data, int advice) => -1;

int $syscallMlock(Slice<GoInt> data) => -1;
int $syscallMunlock(Slice<GoInt> data) => -1;

int $syscallMprotect(Slice<GoInt> data, int prot) => -1;

int $syscallMunmap(Slice<GoInt> data) => -1;

int $syscallShutdown(int fd, int how) => -1;

int $syscallFlock(int fd, int how) => -1;

int $syscallListen(int fd, int how) => -1;

int $syscallSysinfo(Slice<GoInt> data) => 38;

int $syscallGettid() => throw UnsupportedError('syscall.Gettid requires native Linux');

(int, int) $syscallTimes(Slice<GoInt> data) => (-1, 38);

(int, int) $syscallInotifyInit() => (-1, 38);

(int, int) $syscallInotifyInit1(int flags) => (-1, 38);

(int, int) $syscallInotifyAddWatch(int fd, GoString path, int mask) => (-1, 38);

(int, int) $syscallInotifyRmWatch(int fd, int watchdesc) => (-1, 38);

(int, int) $syscallEpollCreate1(int flag) => (-1, 38);

int $syscallEpollCtl(int epfd, int op, int fd, Slice<GoInt> data) => 38;

(int, int) $syscallEpollWait(int epfd, Slice<GoInt> data, int count, int msec) => (-1, 38);

(int, int, int) $syscallPipe() => (0, 0, -1);

(int, int, int) $syscallPipe2(int flags) => (0, 0, 38);

(int, int) $syscallSplice(int rfd, Slice<GoInt> read, int wfd, Slice<GoInt> write, int length, int flags) => (-1, 38);

(int, int) $syscallTee(int rfd, int wfd, int length, int flags) => (-1, 38);

(int, int) $syscallDarwinSendfile(int outfd, int infd, int offset, int count) =>
    (0, -1);

(int, int) $syscallSendfile(int outfd, int infd, Slice<GoInt> offset, int count) => (-1, 38);

int $syscallNanosleep(Slice<GoInt> request, Slice<GoInt> leftover) => 38;

void $syscallSync() => throw UnsupportedError('syscall.Sync');

int $syscallSyncFileRange(int fd, int offset, int length, int flags) => 38;

int $syscallExchangedata(GoString first, GoString second, int options) => 78;

int $syscallRevoke(GoString path) => -1;
int $syscallUndelete(GoString path) => -1;

(int, int) $syscallKevent(int kq, Slice<GoInt> changes, Slice<GoInt> events, Slice<GoInt> timeout) => (-1, 78);

(int, int) $syscallKqueue() => (-1, -1);

int $syscallGetdtablesize() => throw UnsupportedError('syscall.Getdtablesize');
bool $syscallIssetugid() => throw UnsupportedError('syscall.Issetugid');

int $syscallDarwinMknod(GoString path, int mode, int dev) => 78;

int $syscallMknodat(int parent, GoString path, int mode, int dev) => 38;

int $syscallFallocate(int fd, int mode, int offset, int length) => 38;

(int, int) $syscallSelect(
  int nfd,
  Slice<GoInt> read,
  Slice<GoInt> write,
  Slice<GoInt> except,
  Slice<GoInt> timeout,
) => (-1, -1);

int $syscallBind(int fd, Slice<GoInt> address) => -1;
int $syscallConnect(int fd, Slice<GoInt> address) => -1;
int $syscallSendto(int fd, Slice<GoInt> data, int flags, Slice<GoInt> address) => -1;

int $syscallGetsockname(int fd, Slice<GoInt> data) => -1;
int $syscallGetpeername(int fd, Slice<GoInt> data) => -1;

(int, int) $syscallRecvfrom(int fd, Slice<GoInt> data, int flags, Slice<GoInt> address) => (-1, -1);

(int, int, int) $syscallAccept(int fd, int flags, Slice<GoInt> address) => (-1, 0, -1);

(int, int) $syscallSendmsgN(int fd, Slice<GoInt> data, Slice<GoInt> control, int flags, Slice<GoInt> address) => (0, -1);
(int, int, int, int) $syscallRecvmsg(int fd, Slice<GoInt> data, Slice<GoInt> control, int flags, Slice<GoInt> address) => (0, 0, 0, -1);
