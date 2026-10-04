// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && linux && (amd64 || arm64)

package syscall

import "syscall/dart"

// Openat opens a path relative to dirfd. The caller owns the descriptor.
func Openat(dirfd int, path string, flags int, mode uint32) (int, error) {
	fd, code := dart.SyscallOpenat(dirfd, path, flags|O_LARGEFILE, mode)
	if code != 0 {
		return fd, Errno(code)
	}
	return fd, nil
}

func Mkdirat(dirfd int, path string, mode uint32) error {
	if code := dart.SyscallMkdirat(dirfd, path, mode); code != 0 {
		return Errno(code)
	}
	return nil
}

func Fchownat(dirfd int, path string, uid, gid, flags int) error {
	if code := dart.SyscallFchownat(dirfd, path, uid, gid, flags); code != 0 {
		return Errno(code)
	}
	return nil
}

func Unlinkat(dirfd int, path string) error {
	if code := dart.SyscallUnlinkat(dirfd, path); code != 0 {
		return Errno(code)
	}
	return nil
}

func Renameat(olddirfd int, oldpath string, newdirfd int, newpath string) error {
	if code := dart.SyscallRenameat(olddirfd, oldpath, newdirfd, newpath); code != 0 {
		return Errno(code)
	}
	return nil
}
