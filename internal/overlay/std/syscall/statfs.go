// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && (darwin || linux)

package syscall

import (
	errorspkg "errors"
	"syscall/dart"
)

func Statfs(path string, stat *Statfs_t) error {
	data, code := dart.SyscallStatfs(path, stat != nil)
	return storeStatfs(data, code, stat)
}

func Fstatfs(fd int, stat *Statfs_t) error {
	data, code := dart.SyscallFstatfs(fd, stat != nil)
	return storeStatfs(data, code, stat)
}

func storeStatfs(data []int64, code int, stat *Statfs_t) error {
	if code == -1 {
		return errorspkg.ErrUnsupported
	}
	if code != 0 {
		return Errno(code)
	}
	*stat = statfsFromData(data)
	return nil
}
