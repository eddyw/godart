// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && linux && (amd64 || arm64)

package syscall

import (
	"slices"
	"syscall/dart"
)

func Faccessat(dirfd int, path string, mode uint32, flags int) error {
	errno := Errno(dart.SyscallFaccessat(dirfd, path, mode, flags))
	if errno == 0 {
		return nil
	}
	if flags == 0 || errno != ENOSYS && errno != EPERM {
		return errno
	}
	const noFollow = 0x100
	const effective = 0x200
	if flags & ^(noFollow|effective) != 0 {
		return EINVAL
	}
	fileMode, owner, group, code := dart.SyscallFaccessatStat(dirfd, path, flags&noFollow)
	if code != 0 {
		return Errno(code)
	}
	mode &= 7
	if mode == 0 {
		return nil
	}
	var uid int
	if flags&effective != 0 {
		uid = Geteuid()
		if uid != 0 && dart.SyscallCapDacOverride() {
			uid = 0
		}
	} else {
		uid = Getuid()
	}
	if uid == 0 {
		if mode&1 == 0 || fileMode&0111 != 0 {
			return nil
		}
		return EACCES
	}
	var permitted uint32
	if uint32(uid) == owner {
		permitted = (fileMode >> 6) & 7
	} else {
		var gid int
		if flags&effective != 0 {
			gid = Getegid()
		} else {
			gid = Getgid()
		}
		member := uint32(gid) == group
		if !member {
			groups, err := Getgroups()
			member = err == nil && slices.Contains(groups, int(group))
		}
		if member {
			permitted = (fileMode >> 3) & 7
		} else {
			permitted = fileMode & 7
		}
	}
	if permitted&mode == mode {
		return nil
	}
	return EACCES
}
