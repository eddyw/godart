// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && linux && amd64

package syscall

import "syscall/dart"

func Utime(path string, buf *Utimbuf) error {
	var atime, mtime int64
	if buf != nil {
		atime, mtime = buf.Actime, buf.Modtime
	}
	if code := dart.SyscallUtime(path, buf != nil, atime, mtime); code != 0 {
		return Errno(code)
	}
	return nil
}

func utimes(path string, tv []Timeval) error {
	if code := dart.SyscallUtimes(path, tv[0].Sec, tv[0].Usec, tv[1].Sec, tv[1].Usec); code != 0 {
		return Errno(code)
	}
	return nil
}

func futimesat(dirfd int, path string, tv []Timeval) error {
	if code := dart.SyscallFutimesat(dirfd, path, tv[0].Sec, tv[0].Usec, tv[1].Sec, tv[1].Usec); code != 0 {
		return Errno(code)
	}
	return nil
}
