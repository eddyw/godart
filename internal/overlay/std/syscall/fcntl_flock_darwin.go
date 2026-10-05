// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && (amd64 || arm64)

package syscall

import "syscall/dart"

// FcntlFlock performs a fcntl syscall for the [F_GETLK], [F_SETLK] or [F_SETLKW] command.
func FcntlFlock(fd uintptr, cmd int, lk *Flock_t) error {
	var start, length int64
	var pid int32
	var kind, whence int16
	var pad0, pad1 uint32
	if lk != nil {
		start, length, pid, kind, whence = lk.Start, lk.Len, lk.Pid, lk.Type, lk.Whence

	}
	data, errno := dart.SyscallFcntlFlock(fd, cmd, lk != nil, start, length, pid, kind, whence, pad0, pad1)
	if errno == -1 {
		return syscallErrno(errno)
	}
	if lk != nil {
		lk.Type, lk.Whence = int16(data[0]), int16(data[1])
		lk.Start, lk.Len, lk.Pid = data[2], data[3], int32(data[4])

	}
	if errno != 0 {
		return Errno(errno)
	}
	return nil
}
