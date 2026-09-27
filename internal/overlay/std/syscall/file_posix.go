//go:build dart && (linux || darwin) && (amd64 || arm64)

package syscall

import "syscall/dart"

func Open(path string, mode int, perm uint32) (int, error) {
	fd, errno := dart.SyscallOpen(path, mode, perm)
	if errno != 0 {
		return fd, Errno(errno)
	}
	return fd, nil
}

func Close(fd int) error {
	if errno := dart.SyscallClose(fd); errno != 0 {
		return Errno(errno)
	}
	return nil
}

func ReadDirent(fd int, buf []byte) (int, error) {
	n, errno := dart.SyscallReadDirent(fd, buf)
	if errno != 0 {
		return n, Errno(errno)
	}
	return n, nil
}

func Seek(fd int, offset int64, whence int) (int64, error) {
	position, errno := dart.SyscallSeek(fd, offset, whence)
	if errno != 0 {
		return position, Errno(errno)
	}
	return position, nil
}

func Chdir(path string) error {
	if errno := dart.NativeChdir(path); errno != 0 {
		return Errno(errno)
	}
	return nil
}

func Chmod(path string, mode uint32) error {
	if errno := dart.NativeChmod(path, int(mode)); errno != 0 {
		return Errno(errno)
	}
	return nil
}

func Chown(path string, uid, gid int) error {
	if errno := dart.NativeChown(path, uid, gid, true); errno != 0 {
		return Errno(errno)
	}
	return nil
}

func Lchown(path string, uid, gid int) error {
	if errno := dart.NativeChown(path, uid, gid, false); errno != 0 {
		return Errno(errno)
	}
	return nil
}

func Link(oldpath, newpath string) error {
	if errno := dart.NativeLink(oldpath, newpath); errno != 0 {
		return Errno(errno)
	}
	return nil
}

func Symlink(oldpath, newpath string) error {
	if errno := dart.NativeSymlink(oldpath, newpath); errno != 0 {
		return Errno(errno)
	}
	return nil
}

func Rename(oldpath, newpath string) error {
	if errno := dart.NativeRename(oldpath, newpath); errno != 0 {
		return Errno(errno)
	}
	return nil
}

func Truncate(path string, length int64) error {
	if errno := dart.NativeTruncate(path, length); errno != 0 {
		return Errno(errno)
	}
	return nil
}
