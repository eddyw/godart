// Copyright 2009,2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package syscall

import (
	errorspkg "errors"
	"syscall/dart"
)

func Pathconf(path string, name int) (int, error) {
	value, code := dart.SyscallPathconf(path, name)
	if code == -1 {
		return 0, errorspkg.ErrUnsupported
	}
	if code != 0 {
		return value, Errno(code)
	}
	return value, nil
}

func Fpathconf(fd int, name int) (int, error) {
	value, code := dart.SyscallFpathconf(fd, name)
	if code == -1 {
		return 0, errorspkg.ErrUnsupported
	}
	if code != 0 {
		return value, Errno(code)
	}
	return value, nil
}
