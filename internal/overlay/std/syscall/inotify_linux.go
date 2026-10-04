// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && linux && (amd64 || arm64)

package syscall

import (
	"strings"
	"syscall/dart"
)

func InotifyInit() (int, error) {
	fd, code := dart.SyscallInotifyInit()
	if code != 0 {
		return fd, Errno(code)
	}
	return fd, nil
}

func InotifyInit1(flags int) (int, error) {
	fd, code := dart.SyscallInotifyInit1(flags)
	if code != 0 {
		return fd, Errno(code)
	}
	return fd, nil
}

func InotifyAddWatch(fd int, pathname string, mask uint32) (int, error) {
	if strings.IndexByte(pathname, 0) >= 0 {
		return 0, EINVAL
	}
	wd, code := dart.SyscallInotifyAddWatch(fd, pathname, mask)
	if code != 0 {
		return wd, Errno(code)
	}
	return wd, nil
}

func InotifyRmWatch(fd int, watchdesc uint32) (int, error) {
	success, code := dart.SyscallInotifyRmWatch(fd, watchdesc)
	if code != 0 {
		return success, Errno(code)
	}
	return success, nil
}
