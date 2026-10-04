// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && darwin && (amd64 || arm64)

package syscall

import "syscall/dart"

func GetsockoptByte(fd, level, opt int) (byte, error) {
	value, code := dart.SyscallGetsockoptByte(fd, level, opt)
	if code != 0 {
		return byte(value), Errno(code)
	}
	return byte(value), nil
}
