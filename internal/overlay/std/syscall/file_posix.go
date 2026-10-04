// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && (linux || darwin) && (amd64 || arm64)

package syscall

import "syscall/dart"

func Open(path string, mode int, perm uint32) (int, error) {
	fd, errno := dart.SyscallOpen(path, mode, perm)
	if errno != 0 {
		return fd, syscallErrno(errno)
	}
	return fd, nil
}

func Close(fd int) error {
	if errno := dart.SyscallClose(fd); errno != 0 {
		return syscallErrno(errno)
	}
	return nil
}

func Dup(oldfd int) (int, error) {
	fd, errno := dart.SyscallDup(oldfd)
	if errno != 0 {
		return fd, syscallErrno(errno)
	}
	return fd, nil
}

func Dup2(oldfd, newfd int) error {
	if errno := dart.SyscallDup2(oldfd, newfd); errno != 0 {
		return syscallErrno(errno)
	}
	return nil
}

func ReadDirent(fd int, buf []byte) (int, error) {
	n, errno := dart.SyscallReadDirent(fd, buf)
	if errno != 0 {
		return n, syscallErrno(errno)
	}
	return n, nil
}

func Seek(fd int, offset int64, whence int) (int64, error) {
	position, errno := dart.SyscallSeek(fd, offset, whence)
	if errno != 0 {
		return position, syscallErrno(errno)
	}
	return position, nil
}

func Chdir(path string) error {
	if errno := dart.NativeChdir(path); errno != 0 {
		return syscallErrno(errno)
	}
	return nil
}

func Chmod(path string, mode uint32) error {
	if errno := dart.NativeChmod(path, int(mode)); errno != 0 {
		return syscallErrno(errno)
	}
	return nil
}

func Chown(path string, uid, gid int) error {
	if errno := dart.NativeChown(path, uid, gid, true); errno != 0 {
		return syscallErrno(errno)
	}
	return nil
}

func Lchown(path string, uid, gid int) error {
	if errno := dart.NativeChown(path, uid, gid, false); errno != 0 {
		return syscallErrno(errno)
	}
	return nil
}

func Link(oldpath, newpath string) error {
	if errno := dart.NativeLink(oldpath, newpath); errno != 0 {
		return syscallErrno(errno)
	}
	return nil
}

func Symlink(oldpath, newpath string) error {
	if errno := dart.NativeSymlink(oldpath, newpath); errno != 0 {
		return syscallErrno(errno)
	}
	return nil
}

func Rename(oldpath, newpath string) error {
	if errno := dart.NativeRename(oldpath, newpath); errno != 0 {
		return syscallErrno(errno)
	}
	return nil
}

func Truncate(path string, length int64) error {
	if errno := dart.NativeTruncate(path, length); errno != 0 {
		return syscallErrno(errno)
	}
	return nil
}

func Fchdir(fd int) error {
	if errno := dart.DirectoryChdir(fd); errno != 0 {
		return syscallErrno(errno)
	}
	return nil
}

func Fchmod(fd int, mode uint32) error {
	if errno := dart.FileChmod(fd, int(mode)); errno != 0 {
		return syscallErrno(errno)
	}
	return nil
}

func Fchown(fd, uid, gid int) error {
	if errno := dart.FileChown(fd, uid, gid); errno != 0 {
		return syscallErrno(errno)
	}
	return nil
}

func Ftruncate(fd int, length int64) error {
	if errno := dart.NativeFtruncate(fd, length); errno != 0 {
		return syscallErrno(errno)
	}
	return nil
}

func Pread(fd int, p []byte, offset int64) (int, error) {
	n, errno := dart.NativePread(fd, p, offset)
	if errno != 0 {
		return -1, syscallErrno(errno)
	}
	return n, nil
}

func Pwrite(fd int, p []byte, offset int64) (int, error) {
	n, errno := dart.NativePwrite(fd, p, offset)
	if errno != 0 {
		return -1, syscallErrno(errno)
	}
	return n, nil
}

func Mkdir(path string, mode uint32) error {
	if errno := dart.DirectoryCreate(path, int(mode)); errno != 0 {
		return syscallErrno(errno)
	}
	return nil
}

func Unlink(path string) error {
	if errno := dart.SyscallUnlink(path, false); errno != 0 {
		return syscallErrno(errno)
	}
	return nil
}

func Rmdir(path string) error {
	if errno := dart.SyscallUnlink(path, true); errno != 0 {
		return syscallErrno(errno)
	}
	return nil
}

func Readlink(path string, buf []byte) (int, error) {
	n, errno := dart.SyscallReadlink(path, buf)
	if errno != 0 {
		return n, syscallErrno(errno)
	}
	return n, nil
}

func Read(fd int, p []byte) (int, error) {
	n, code := dart.NativeRead(fd, p)
	if code != 0 {
		return n, syscallErrno(code)
	}
	return n, nil
}

func Write(fd int, p []byte) (int, error) {
	n, code := dart.NativeWrite(fd, p)
	if code != 0 {
		return n, syscallErrno(code)
	}
	return n, nil
}

func Flock(fd, how int) error {
	if code := dart.SyscallFlock(fd, how); code != 0 {
		return syscallErrno(code)
	}
	return nil
}
