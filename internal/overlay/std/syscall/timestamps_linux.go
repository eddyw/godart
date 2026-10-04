// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && linux && (amd64 || arm64)

package syscall

import (
	"strconv"
	"syscall/dart"
)

func Utimes(path string, tv []Timeval) error {
	if len(tv) != 2 {
		return EINVAL
	}
	return utimes(path, tv)
}

func UtimesNano(path string, ts []Timespec) error {
	if len(ts) != 2 {
		return EINVAL
	}
	return utimensat(-100, path, ts)
}

func Futimesat(dirfd int, path string, tv []Timeval) error {
	if len(tv) != 2 {
		return EINVAL
	}
	return futimesat(dirfd, path, tv)
}

func Futimes(fd int, tv []Timeval) error {
	return Utimes("/proc/self/fd/"+strconv.Itoa(fd), tv)
}

func utimensat(dirfd int, path string, ts []Timespec) error {
	if code := dart.SyscallUtimensat(dirfd, path, ts[0].Sec, ts[0].Nsec, ts[1].Sec, ts[1].Nsec); code != 0 {
		return syscallErrno(code)
	}
	return nil
}
