// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && linux && (amd64 || arm64)

package syscall

import "syscall/dart"

func Fallocate(fd int, mode uint32, off, length int64) error {
	if code := dart.SyscallFallocate(fd, mode, off, length); code != 0 {
		return Errno(code)
	}
	return nil
}
