// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && darwin && (amd64 || arm64)

package syscall

import "syscall/dart"

func Utimes(path string, tv []Timeval) error {
	if len(tv) != 2 {
		return EINVAL
	}
	if code := dart.SyscallUtimes(path, tv[0].Sec, int64(tv[0].Usec), tv[1].Sec, int64(tv[1].Usec)); code != 0 {
		return syscallErrno(code)
	}
	return nil
}

func UtimesNano(path string, ts []Timespec) error {
	if len(ts) != 2 {
		return EINVAL
	}
	code := dart.SyscallUtimensat(-2, path, ts[0].Sec, ts[0].Nsec, ts[1].Sec, ts[1].Nsec)
	if code == 0 {
		return nil
	}
	if syscallErrno(code) != ENOSYS {
		return syscallErrno(code)
	}
	tv := []Timeval{
		NsecToTimeval(TimespecToNsec(ts[0])),
		NsecToTimeval(TimespecToNsec(ts[1])),
	}
	return Utimes(path, tv)
}

func Futimes(fd int, tv []Timeval) error {
	if len(tv) != 2 {
		return EINVAL
	}
	if code := dart.SyscallFutimes(fd, tv[0].Sec, int64(tv[0].Usec), tv[1].Sec, int64(tv[1].Usec)); code != 0 {
		return syscallErrno(code)
	}
	return nil
}
