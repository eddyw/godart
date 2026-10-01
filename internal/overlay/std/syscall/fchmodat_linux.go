// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && linux && (amd64 || arm64)

package syscall

import "syscall/dart"

func Fchmodat(dirfd int, path string, mode uint32, flags int) error {
	errno := Errno(dart.SyscallFchmodat(dirfd, path, mode, flags))
	if flags != 0 && errno == ENOSYS {
		const supported = 0x100 | 0x1000 // AT_SYMLINK_NOFOLLOW and AT_EMPTY_PATH.
		if flags&^supported != 0 {
			return EINVAL
		}
		return EOPNOTSUPP
	}
	if errno != 0 {
		return errno
	}
	return nil
}
