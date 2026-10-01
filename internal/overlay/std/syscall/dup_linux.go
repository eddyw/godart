// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && linux && (amd64 || arm64)

package syscall

import "syscall/dart"

func Dup3(oldfd, newfd, flags int) error {
	if code := dart.SyscallDup3(oldfd, newfd, flags); code != 0 {
		return Errno(code)
	}
	return nil
}
