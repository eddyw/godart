// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package syscall

import (
	errorspkg "errors"
	"internal/oserror"
	"strconv"
)

// Errno holds an operating system error number.
type Errno uintptr

// The unavailable backend returns -1. Native backends return positive errno values.
func syscallErrno(code int) Errno {
	if code == -1 {
		return ENOSYS
	}
	return Errno(code)
}

func (e Errno) Error() string {
	if 0 <= int(e) && int(e) < len(errors) {
		if text := errors[e]; text != "" {
			return text
		}
	}
	return "errno " + strconv.Itoa(int(e))
}

func (e Errno) Is(target error) bool {
	switch target {
	case oserror.ErrPermission:
		return e == EACCES || e == EPERM
	case oserror.ErrExist:
		return e == EEXIST || e == ENOTEMPTY
	case oserror.ErrNotExist:
		return e == ENOENT
	case errorspkg.ErrUnsupported:
		return e == ENOSYS || e == ENOTSUP || e == EOPNOTSUPP
	}
	return false
}

func (e Errno) Temporary() bool {
	return e == EINTR || e == EMFILE || e == ENFILE || e.Timeout()
}

func (e Errno) Timeout() bool {
	return e == EAGAIN || e == EWOULDBLOCK || e == ETIMEDOUT
}
