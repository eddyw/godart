// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package syscall

import (
	errorspkg "errors"
	"syscall/dart"
)

func Getxattr(path string, attr string, dest []byte) (int, error) {
	size, code := dart.SyscallGetxattr(path, attr, dest)
	return size, xattrError(code)
}

func Listxattr(path string, dest []byte) (int, error) {
	size, code := dart.SyscallListxattr(path, dest)
	return size, xattrError(code)
}

func Setxattr(path string, attr string, data []byte, flags int) error {
	return xattrError(dart.SyscallSetxattr(path, attr, data, flags))
}

func Removexattr(path string, attr string) error {
	return xattrError(dart.SyscallRemovexattr(path, attr))
}

func xattrError(code int) error {
	if code == -1 {
		return errorspkg.ErrUnsupported
	}
	if code != 0 {
		return Errno(code)
	}
	return nil
}
